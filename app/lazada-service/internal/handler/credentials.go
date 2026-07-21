package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

const (
	headerLazadaAppKey    = "X-Lazada-App-Key"
	headerLazadaAppSecret = "X-Lazada-App-Secret"
)

func requireLazadaCredentials(c *fiber.Ctx) (client.Credentials, error) {
	creds := client.Credentials{
		AppKey:    c.Get(headerLazadaAppKey),
		AppSecret: c.Get(headerLazadaAppSecret),
	}
	if creds.AppKey == "" {
		return creds, errors.New("X-Lazada-App-Key header is required")
	}
	if creds.AppSecret == "" {
		return creds, errors.New("X-Lazada-App-Secret header is required")
	}
	return creds, nil
}

func badRequestCredentials(c *fiber.Ctx, err error) error {
	return utils.BadRequest(c, err)
}
