package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/viber-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

const headerViberSignature = "X-Viber-Content-Signature"

type WebhookHandler struct {
	svc service.WebhookService
}

func NewWebhookHandler(svc service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()
	signature := c.Get(headerViberSignature)
	authToken := c.Query("auth_token")
	callbackURL := c.Query("callback_url")

	if err := h.svc.Handle(c.Context(), signature, body, authToken, callbackURL); err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}
