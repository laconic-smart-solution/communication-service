package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/telegram-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/telegram-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/telegram-service/internal/response"
)

// Telegram has no multicast API; we send one-by-one and group for rate pacing.
const sendChunkSize = 30

type BatchConfig struct {
	MinInterval   time.Duration
	MaxRecipients int
	MaxRetries    int
	BackoffBase   time.Duration
}

func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		MinInterval:   50 * time.Millisecond,
		MaxRecipients: 10000,
		MaxRetries:    3,
		BackoffBase:   time.Second,
	}
}

type MessageService interface {
	Send(ctx context.Context, botToken string, req request.SendMessageRequest) (*response.SendMessageResponse, error)
	SendBatch(ctx context.Context, botToken string, req request.SendBatchMessageRequest) (*response.SendBatchMessageResponse, error)
}

type messageServiceImpl struct {
	logger   *logrus.Logger
	telegram client.TelegramClient
	batch    BatchConfig
	sleep    func(context.Context, time.Duration) error
}

var _ MessageService = (*messageServiceImpl)(nil)

func NewMessageService(logger *logrus.Logger, telegram client.TelegramClient, batch BatchConfig) MessageService {
	if batch.MaxRecipients <= 0 {
		batch.MaxRecipients = 10000
	}
	if batch.MaxRetries <= 0 {
		batch.MaxRetries = 3
	}
	if batch.BackoffBase <= 0 {
		batch.BackoffBase = time.Second
	}
	return &messageServiceImpl{
		logger:   logger,
		telegram: telegram,
		batch:    batch,
		sleep:    sleepContext,
	}
}

func (s *messageServiceImpl) Send(ctx context.Context, botToken string, req request.SendMessageRequest) (*response.SendMessageResponse, error) {
	if botToken == "" {
		return nil, errors.New("X-Telegram-Bot-Token header is required")
	}

	err := s.telegram.SendMessage(ctx, client.SendMessageInput{
		BotToken:    botToken,
		RecipientID: req.RecipientID,
		Text:        req.Text,
	})
	if err != nil {
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("telegram rate limit exceeded")
		}
		if errors.Is(err, client.ErrInvalidToken) {
			return nil, errors.New("invalid telegram bot token")
		}
		s.logger.WithError(err).Error("send message")
		return nil, errors.New("failed to send message")
	}

	return &response.SendMessageResponse{
		RecipientID: req.RecipientID,
		Status:      "sent",
	}, nil
}

func (s *messageServiceImpl) SendBatch(ctx context.Context, botToken string, req request.SendBatchMessageRequest) (*response.SendBatchMessageResponse, error) {
	if botToken == "" {
		return nil, errors.New("X-Telegram-Bot-Token header is required")
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, errors.New("text is required")
	}

	ids := dedupeRecipientIDs(req.RecipientIDs)
	if len(ids) == 0 {
		return nil, errors.New("recipient_ids is required")
	}
	if len(ids) > s.batch.MaxRecipients {
		return nil, fmt.Errorf("recipient_ids exceeds maximum of %d", s.batch.MaxRecipients)
	}

	chunks := chunkStrings(ids, sendChunkSize)
	out := &response.SendBatchMessageResponse{
		Total:      len(ids),
		ChunkCount: len(chunks),
	}

	for i, chunk := range chunks {
		if i > 0 && s.batch.MinInterval > 0 {
			if err := s.sleep(ctx, s.batch.MinInterval); err != nil {
				return nil, err
			}
		}

		for _, recipientID := range chunk {
			err := s.sendWithRetry(ctx, botToken, recipientID, text)
			if err != nil {
				if errors.Is(err, client.ErrInvalidToken) {
					return nil, errors.New("invalid telegram bot token")
				}
				s.logger.WithError(err).WithField("recipient_id", recipientID).Error("send batch recipient")
				out.Failed++
				out.FailedRecipientIDs = append(out.FailedRecipientIDs, recipientID)
				continue
			}
			out.Sent++
			if s.batch.MinInterval > 0 {
				if err := s.sleep(ctx, s.batch.MinInterval); err != nil {
					return nil, err
				}
			}
		}
	}

	return out, nil
}

func (s *messageServiceImpl) sendWithRetry(ctx context.Context, botToken, recipientID, text string) error {
	var lastErr error
	for attempt := 0; attempt <= s.batch.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := s.batch.BackoffBase * time.Duration(1<<(attempt-1))
			if err := s.sleep(ctx, backoff); err != nil {
				return err
			}
		}

		err := s.telegram.SendMessage(ctx, client.SendMessageInput{
			BotToken:    botToken,
			RecipientID: recipientID,
			Text:         text,
		})
		if err == nil {
			return nil
		}
		lastErr = err
		if !errors.Is(err, client.ErrRateLimited) {
			return err
		}
	}
	return lastErr
}

func dedupeRecipientIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func chunkStrings(ids []string, size int) [][]string {
	if size <= 0 {
		size = sendChunkSize
	}
	if len(ids) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(ids)+size-1)/size)
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[i:end])
	}
	return chunks
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
