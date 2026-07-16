package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/client"
)

type WebhookService interface {
	Handle(ctx context.Context, signature string, body []byte, channelSecret, callbackURL string) error
}

type webhookServiceImpl struct {
	logger *logrus.Logger
	line   client.LINEClient
}

var _ WebhookService = (*webhookServiceImpl)(nil)

func NewWebhookService(logger *logrus.Logger, line client.LINEClient) WebhookService {
	return &webhookServiceImpl{logger: logger, line: line}
}

func (s *webhookServiceImpl) Handle(ctx context.Context, signature string, body []byte, channelSecret, callbackURL string) error {
	if strings.TrimSpace(channelSecret) == "" {
		return errors.New("channel_secret query parameter is required")
	}

	if err := s.line.VerifyWebhook(signature, body, channelSecret); err != nil {
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

	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" {
		return nil
	}

	parsed, err := url.Parse(callbackURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("callback_url must be a valid https URL")
	}

	if err := s.line.ForwardEvents(ctx, callbackURL, events); err != nil {
		s.logger.WithError(err).WithField("callback_url", parsed.Host).Error("forward webhook events")
		return errors.New("failed to forward webhook events")
	}
	return nil
}
