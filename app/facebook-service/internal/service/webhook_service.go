package service

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/config"
)

type WebhookService interface {
	VerifyChallenge(mode, token, challenge string) (string, error)
	Handle(ctx context.Context, signature string, body []byte) error
}

type webhookServiceImpl struct {
	logger *logrus.Logger
	cfg    *config.Config
	graph  client.GraphClient
}

var _ WebhookService = (*webhookServiceImpl)(nil)

func NewWebhookService(logger *logrus.Logger, cfg *config.Config, graph client.GraphClient) WebhookService {
	return &webhookServiceImpl{logger: logger, cfg: cfg, graph: graph}
}

func (s *webhookServiceImpl) VerifyChallenge(mode, token, challenge string) (string, error) {
	if mode != "subscribe" {
		return "", errors.New("invalid hub mode")
	}
	if token != s.cfg.FacebookVerifyToken {
		return "", errors.New("invalid verify token")
	}
	if challenge == "" {
		return "", errors.New("missing hub challenge")
	}
	return challenge, nil
}

func (s *webhookServiceImpl) Handle(ctx context.Context, signature string, body []byte) error {
	if err := s.graph.VerifyWebhook(signature, body, s.cfg.FacebookAppSecret); err != nil {
		if errors.Is(err, client.ErrInvalidSignature) {
			return errors.New("invalid webhook signature")
		}
		return err
	}
	s.logger.WithField("bytes", len(body)).Info("facebook webhook received")
	return nil
}
