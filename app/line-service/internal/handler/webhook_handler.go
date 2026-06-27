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

	if err := h.svc.Handle(c.Context(), signature, body); err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}
