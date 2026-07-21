package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/whatsapp-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

const headerHubSignature = "X-Hub-Signature-256"

type WebhookHandler struct {
	svc service.WebhookService
}

func NewWebhookHandler(svc service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// Verify handles Meta Cloud API webhook subscription handshake (GET).
func (h *WebhookHandler) Verify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")
	expected := c.Query("verify_token")
	if expected == "" {
		expected = c.Query("expected_verify_token")
	}

	body, err := h.svc.VerifySubscription(mode, token, challenge, expected)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	c.Set(fiber.HeaderContentType, "text/plain")
	return c.SendString(body)
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()
	signature := c.Get(headerHubSignature)
	appSecret := c.Query("app_secret")
	callbackURL := c.Query("callback_url")

	if err := h.svc.Handle(c.Context(), signature, body, appSecret, callbackURL); err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}
