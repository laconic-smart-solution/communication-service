package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

const (
	headerTikTokAppKey    = "X-TikTok-App-Key"
	headerTikTokAppSecret = "X-TikTok-App-Secret"
)

func requireTikTokCredentials(c *fiber.Ctx) (client.Credentials, error) {
	creds := client.Credentials{
		AppKey:    c.Get(headerTikTokAppKey),
		AppSecret: c.Get(headerTikTokAppSecret),
	}
	if creds.AppKey == "" {
		return creds, errors.New("X-TikTok-App-Key header is required")
	}
	if creds.AppSecret == "" {
		return creds, errors.New("X-TikTok-App-Secret header is required")
	}
	return creds, nil
}

func badRequestCredentials(c *fiber.Ctx, err error) error {
	return utils.BadRequest(c, err)
}
