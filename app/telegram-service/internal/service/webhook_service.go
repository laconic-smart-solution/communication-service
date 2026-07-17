package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/telegram-service/internal/client"
)

type WebhookService interface {
	Handle(ctx context.Context, secretHeader string, body []byte, secretToken, callbackURL string) error
}

type webhookServiceImpl struct {
	logger   *logrus.Logger
	telegram client.TelegramClient
}

var _ WebhookService = (*webhookServiceImpl)(nil)

func NewWebhookService(logger *logrus.Logger, telegram client.TelegramClient) WebhookService {
	return &webhookServiceImpl{logger: logger, telegram: telegram}
}

func (s *webhookServiceImpl) Handle(ctx context.Context, secretHeader string, body []byte, secretToken, callbackURL string) error {
	if strings.TrimSpace(secretToken) == "" {
		return errors.New("secret_token query parameter is required")
	}

	if err := s.telegram.VerifyWebhook(secretHeader, secretToken); err != nil {
		if errors.Is(err, client.ErrInvalidSignature) {
			return errors.New("invalid webhook secret token")
		}
		return err
	}

	events, err := s.telegram.ParseWebhook(body)
	if err != nil {
		s.logger.WithError(err).Error("parse webhook")
		return errors.New("failed to parse webhook")
	}

	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" || len(events) == 0 {
		return nil
	}

	parsed, err := url.Parse(callbackURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("callback_url must be a valid https URL")
	}

	if err := s.telegram.ForwardEvents(ctx, callbackURL, events); err != nil {
		s.logger.WithError(err).WithField("callback_url", parsed.Host).Error("forward webhook events")
		return errors.New("failed to forward webhook events")
	}
	return nil
}
