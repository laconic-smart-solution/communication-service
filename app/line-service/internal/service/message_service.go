package service

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/response"
)

type MessageService interface {
	Send(ctx context.Context, accessToken string, req request.SendMessageRequest) (*response.SendMessageResponse, error)
}

type messageServiceImpl struct {
	logger *logrus.Logger
	line   client.LINEClient
}

var _ MessageService = (*messageServiceImpl)(nil)

func NewMessageService(logger *logrus.Logger, line client.LINEClient) MessageService {
	return &messageServiceImpl{logger: logger, line: line}
}

func (s *messageServiceImpl) Send(ctx context.Context, accessToken string, req request.SendMessageRequest) (*response.SendMessageResponse, error) {
	if accessToken == "" {
		return nil, errors.New("X-Line-Channel-Access-Token header is required")
	}

	err := s.line.SendMessage(ctx, client.SendMessageInput{
		AccessToken: accessToken,
		RecipientID: req.RecipientID,
		Text:        req.Text,
	})
	if err != nil {
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("line rate limit exceeded")
		}
		if errors.Is(err, client.ErrInvalidToken) {
			return nil, errors.New("invalid line channel token")
		}
		s.logger.WithError(err).Error("send message")
		return nil, errors.New("failed to send message")
	}

	return &response.SendMessageResponse{
		RecipientID: req.RecipientID,
		Status:      "sent",
	}, nil
}
