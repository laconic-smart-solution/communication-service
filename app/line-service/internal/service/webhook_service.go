package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/config"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/repository"
)

type WebhookService interface {
	Handle(ctx context.Context, signature string, body []byte) error
}

type webhookServiceImpl struct {
	logger   *logrus.Logger
	cfg      *config.Config
	line     client.LINEClient
	messages repository.MessageRepository
	channels repository.ChannelRepository
}

var _ WebhookService = (*webhookServiceImpl)(nil)

func NewWebhookService(
	logger *logrus.Logger,
	cfg *config.Config,
	line client.LINEClient,
	messages repository.MessageRepository,
	channels repository.ChannelRepository,
) WebhookService {
	return &webhookServiceImpl{logger: logger, cfg: cfg, line: line, messages: messages, channels: channels}
}

func (s *webhookServiceImpl) Handle(ctx context.Context, signature string, body []byte) error {
	if err := s.line.VerifyWebhook(signature, body, s.cfg.LineChannelSecret); err != nil {
		if errors.Is(err, client.ErrInvalidSignature) {
			return errors.New("invalid webhook signature")
		}
		return err
	}

	events, err := s.line.ParseWebhook(body)
	if err != nil {
		s.logger.WithError(err).Error("parse webhook")
		return errors.New("failed to parse webhook")
	}

	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	for _, ev := range events {
		msg := &models.Message{
			ChannelID:   uuid.Nil,
			TenantID:    tenantID,
			Direction:   models.MessageDirectionInbound,
			RecipientID: ev.UserID,
			Body:        ev.Text,
			Status:      models.MessageStatusSent,
			ExternalID:  ev.MessageID,
		}
		if err := s.messages.Create(msg); err != nil {
			s.logger.WithError(err).Error("persist inbound message")
		}
	}

	return nil
}
