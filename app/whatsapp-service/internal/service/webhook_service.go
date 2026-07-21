package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/whatsapp-service/internal/client"
)

type WebhookService interface {
	VerifySubscription(mode, token, challenge, expectedToken string) (string, error)
	Handle(ctx context.Context, signatureHeader string, body []byte, appSecret, callbackURL string) error
}

type webhookServiceImpl struct {
	logger    *logrus.Logger
	whatsapp  client.WhatsAppClient
}

var _ WebhookService = (*webhookServiceImpl)(nil)

func NewWebhookService(logger *logrus.Logger, whatsapp client.WhatsAppClient) WebhookService {
	return &webhookServiceImpl{logger: logger, whatsapp: whatsapp}
}

func (s *webhookServiceImpl) VerifySubscription(mode, token, challenge, expectedToken string) (string, error) {
	if mode != "subscribe" {
		return "", errors.New("hub.mode must be subscribe")
	}
	expectedToken = strings.TrimSpace(expectedToken)
	if expectedToken == "" {
		return "", errors.New("verify_token query parameter is required")
	}
	if strings.TrimSpace(token) != expectedToken {
		return "", errors.New("hub.verify_token mismatch")
	}
	if strings.TrimSpace(challenge) == "" {
		return "", errors.New("hub.challenge is required")
	}
	return challenge, nil
}

func (s *webhookServiceImpl) Handle(ctx context.Context, signatureHeader string, body []byte, appSecret, callbackURL string) error {
	if strings.TrimSpace(appSecret) == "" {
		return errors.New("app_secret query parameter is required")
	}

	if err := s.whatsapp.VerifyWebhookSignature(signatureHeader, body, appSecret); err != nil {
		if errors.Is(err, client.ErrInvalidSignature) {
			return errors.New("invalid webhook signature")
		}
		return err
	}

	events, err := s.whatsapp.ParseWebhook(body)
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

	if err := s.whatsapp.ForwardEvents(ctx, callbackURL, events); err != nil {
		s.logger.WithError(err).WithField("callback_url", parsed.Host).Error("forward webhook events")
		return errors.New("failed to forward webhook events")
	}
	return nil
}
