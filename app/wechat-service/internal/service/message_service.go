package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/response"
)

type BatchConfig struct {
	MinInterval   time.Duration
	MaxRecipients int
	MaxRetries    int
	BackoffBase   time.Duration
}

type MessageService interface {
	Send(ctx context.Context, accessToken string, req request.SendMessageRequest) (*response.SendMessageResponse, error)
	SendBatch(ctx context.Context, accessToken string, req request.SendBatchMessageRequest) (*response.SendBatchMessageResponse, error)
}

type messageServiceImpl struct {
	logger  *logrus.Logger
	wechat  client.WeChatClient
	batch   BatchConfig
	sleep   func(context.Context, time.Duration) error
}

func NewMessageService(logger *logrus.Logger, wechat client.WeChatClient, batch BatchConfig) MessageService {
	if batch.MaxRecipients <= 0 {
		batch.MaxRecipients = 500
	}
	if batch.MaxRetries <= 0 {
		batch.MaxRetries = 3
	}
	if batch.BackoffBase <= 0 {
		batch.BackoffBase = time.Second
	}
	return &messageServiceImpl{
		logger: logger,
		wechat: wechat,
		batch:  batch,
		sleep:  sleepContext,
	}
}

func (s *messageServiceImpl) Send(ctx context.Context, accessToken string, req request.SendMessageRequest) (*response.SendMessageResponse, error) {
	if accessToken == "" {
		return nil, errors.New("X-WeChat-Access-Token header is required")
	}
	err := s.wechat.SendMessage(ctx, client.SendMessageInput{
		AccessToken: accessToken,
		RecipientID: req.RecipientID,
		Text:        req.Text,
	})
	if err != nil {
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("wechat rate limit exceeded")
		}
		if errors.Is(err, client.ErrInvalidToken) {
			return nil, errors.New("invalid wechat access token")
		}
		s.logger.WithError(err).Error("send message")
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	return &response.SendMessageResponse{
		RecipientID: req.RecipientID,
		Status:      "sent",
	}, nil
}

func (s *messageServiceImpl) SendBatch(ctx context.Context, accessToken string, req request.SendBatchMessageRequest) (*response.SendBatchMessageResponse, error) {
	if accessToken == "" {
		return nil, errors.New("X-WeChat-Access-Token header is required")
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, errors.New("text is required")
	}
	ids := make([]string, 0, len(req.RecipientIDs))
	for _, id := range req.RecipientIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, errors.New("recipient_ids is required")
	}
	if len(ids) > s.batch.MaxRecipients {
		return nil, fmt.Errorf("too many recipients (max %d)", s.batch.MaxRecipients)
	}

	out := &response.SendBatchMessageResponse{
		Total:              len(ids),
		FailedRecipientIDs: []string{},
	}
	for i, id := range ids {
		if i > 0 && s.batch.MinInterval > 0 {
			_ = s.sleep(ctx, s.batch.MinInterval)
		}
		var lastErr error
		for attempt := 0; attempt <= s.batch.MaxRetries; attempt++ {
			lastErr = s.wechat.SendMessage(ctx, client.SendMessageInput{
				AccessToken: accessToken,
				RecipientID: id,
				Text:        text,
			})
			if lastErr == nil {
				out.Sent++
				break
			}
			if errors.Is(lastErr, client.ErrRateLimited) && attempt < s.batch.MaxRetries {
				_ = s.sleep(ctx, s.batch.BackoffBase*time.Duration(attempt+1))
				continue
			}
			out.Failed++
			out.FailedRecipientIDs = append(out.FailedRecipientIDs, id)
			break
		}
	}
	out.ChunkCount = 1
	return out, nil
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
