package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/telegram-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

const headerTelegramSecretToken = "X-Telegram-Bot-Api-Secret-Token"

type WebhookHandler struct {
	svc service.WebhookService
}

func NewWebhookHandler(svc service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()
	secretHeader := c.Get(headerTelegramSecretToken)
	secretToken := c.Query("secret_token")
	callbackURL := c.Query("callback_url")

	if err := h.svc.Handle(c.Context(), secretHeader, body, secretToken, callbackURL); err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}
