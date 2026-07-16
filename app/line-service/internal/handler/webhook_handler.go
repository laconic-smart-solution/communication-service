package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type WebhookHandler struct {
	svc service.WebhookService
}

func NewWebhookHandler(svc service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()
	signature := c.Get("X-Line-Signature")
	channelSecret := c.Query("channel_secret")
	callbackURL := c.Query("callback_url")

	if err := h.svc.Handle(c.Context(), signature, body, channelSecret, callbackURL); err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}
