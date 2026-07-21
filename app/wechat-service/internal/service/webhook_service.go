package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/client"
)

type WebhookService interface {
	Handle(ctx context.Context, signature, timestamp, nonce string, body []byte, token, callbackURL string) error
	VerifyURL(signature, timestamp, nonce, echostr, token string) (string, error)
}

type webhookServiceImpl struct {
	logger *logrus.Logger
	wechat client.WeChatClient
}

func NewWebhookService(logger *logrus.Logger, wechat client.WeChatClient) WebhookService {
	return &webhookServiceImpl{logger: logger, wechat: wechat}
}

func (s *webhookServiceImpl) VerifyURL(signature, timestamp, nonce, echostr, token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", errors.New("token query parameter is required")
	}
	if err := s.wechat.VerifyWebhook(signature, timestamp, nonce, token); err != nil {
		if errors.Is(err, client.ErrInvalidSignature) {
			return "", errors.New("invalid webhook signature")
		}
		return "", err
	}
	return echostr, nil
}

func (s *webhookServiceImpl) Handle(ctx context.Context, signature, timestamp, nonce string, body []byte, token, callbackURL string) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("token query parameter is required")
	}
	if err := s.wechat.VerifyWebhook(signature, timestamp, nonce, token); err != nil {
		if errors.Is(err, client.ErrInvalidSignature) {
			return errors.New("invalid webhook signature")
		}
		return err
	}

	events, err := s.wechat.ParseWebhookXML(body)
	if err != nil {
		s.logger.WithError(err).Error("parse webhook")
		return errors.New("failed to parse webhook")
	}

	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL == "" || len(events) == 0 {
		return nil
	}

	parsed, err := url.Parse(callbackURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return errors.New("callback_url must be a valid URL")
	}

	if err := s.wechat.ForwardEvents(ctx, callbackURL, events); err != nil {
		s.logger.WithError(err).Error("forward webhook events")
		return errors.New("failed to forward webhook events")
	}
	return nil
}
