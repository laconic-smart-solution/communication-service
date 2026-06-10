package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/config"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/mapper"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/repository"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/response"
)

type MessageService interface {
	Send(ctx context.Context, tenantID uuid.UUID, req request.SendMessageRequest, actor *uuid.UUID) (*response.MessageResponse, error)
	GetByID(ctx context.Context, tenantID, messageID uuid.UUID) (*response.MessageResponse, error)
}

type messageServiceImpl struct {
	logger   *logrus.Logger
	cfg      *config.Config
	channels repository.ChannelRepository
	repo     repository.MessageRepository
	line     client.LINEClient
}

var _ MessageService = (*messageServiceImpl)(nil)

func NewMessageService(
	logger *logrus.Logger,
	cfg *config.Config,
	channels repository.ChannelRepository,
	repo repository.MessageRepository,
	line client.LINEClient,
) MessageService {
	return &messageServiceImpl{logger: logger, cfg: cfg, channels: channels, repo: repo, line: line}
}

func (s *messageServiceImpl) Send(ctx context.Context, tenantID uuid.UUID, req request.SendMessageRequest, actor *uuid.UUID) (*response.MessageResponse, error) {
	ch, err := s.channels.GetByIDAndTenant(req.ChannelID, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("channel not found")
		}
		return nil, err
	}

	token := ch.AccessToken
	if token == "" {
		token = s.cfg.LineChannelToken
	}

	msg := &models.Message{
		ChannelID:   ch.ID,
		TenantID:    tenantID,
		Direction:   models.MessageDirectionOutbound,
		RecipientID: req.RecipientID,
		Body:        req.Text,
		Status:      models.MessageStatusPending,
	}
	msg.CreatedBy = actor
	msg.UpdatedBy = actor

	if err := s.repo.Create(msg); err != nil {
		s.logger.WithError(err).Error("create message")
		return nil, errors.New("failed to create message")
	}

	err = s.line.SendMessage(ctx, client.SendMessageInput{
		AccessToken: token,
		RecipientID: req.RecipientID,
		Text:        req.Text,
	})
	if err != nil {
		msg.Status = models.MessageStatusFailed
		msg.ProviderError = err.Error()
		_ = s.repo.Update(msg)
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("line rate limit exceeded")
		}
		if errors.Is(err, client.ErrInvalidToken) {
			return nil, errors.New("invalid line channel token")
		}
		s.logger.WithError(err).Error("send message")
		return nil, errors.New("failed to send message")
	}

	msg.Status = models.MessageStatusSent
	if err := s.repo.Update(msg); err != nil {
		s.logger.WithError(err).Error("update message status")
		return nil, errors.New("failed to update message status")
	}

	res := mapper.ToMessageResponse(*msg)
	return &res, nil
}

func (s *messageServiceImpl) GetByID(ctx context.Context, tenantID, messageID uuid.UUID) (*response.MessageResponse, error) {
	msg, err := s.repo.GetByIDAndTenant(messageID, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("message not found")
		}
		return nil, err
	}
	res := mapper.ToMessageResponse(*msg)
	return &res, nil
}
