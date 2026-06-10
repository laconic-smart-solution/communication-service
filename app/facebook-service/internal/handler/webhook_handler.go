package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/pkg/utils"
)

type WebhookHandler struct {
	svc service.WebhookService
}

func NewWebhookHandler(svc service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

func (h *WebhookHandler) Verify(c *fiber.Ctx) error {
	challenge, err := h.svc.VerifyChallenge(
		c.Query("hub.mode"),
		c.Query("hub.verify_token"),
		c.Query("hub.challenge"),
	)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendString(challenge)
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	signature := c.Get("X-Hub-Signature-256")
	if signature == "" {
		signature = c.Get("X-Hub-Signature")
	}
	if err := h.svc.Handle(c.Context(), signature, c.Body()); err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}
