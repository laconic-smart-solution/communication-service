package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/viber-service/internal/client"
)

type WebhookService interface {
	Handle(ctx context.Context, signatureHeader string, body []byte, authToken, callbackURL string) error
}

type webhookServiceImpl struct {
	logger *logrus.Logger
	viber  client.ViberClient
}

var _ WebhookService = (*webhookServiceImpl)(nil)

func NewWebhookService(logger *logrus.Logger, viber client.ViberClient) WebhookService {
	return &webhookServiceImpl{logger: logger, viber: viber}
}

func (s *webhookServiceImpl) Handle(ctx context.Context, signatureHeader string, body []byte, authToken, callbackURL string) error {
	if strings.TrimSpace(authToken) == "" {
		return errors.New("auth_token query parameter is required")
	}

	if err := s.viber.VerifyWebhook(signatureHeader, body, authToken); err != nil {
		if errors.Is(err, client.ErrInvalidSignature) {
			return errors.New("invalid webhook signature")
		}
		return err
	}

	events, err := s.viber.ParseWebhook(body)
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

	if err := s.viber.ForwardEvents(ctx, callbackURL, events); err != nil {
		s.logger.WithError(err).WithField("callback_url", parsed.Host).Error("forward webhook events")
		return errors.New("failed to forward webhook events")
	}
	return nil
}
