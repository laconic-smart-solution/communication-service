package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type WebhookHandler struct {
	svc service.WebhookService
}

func NewWebhookHandler(svc service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// Verify handles WeChat GET URL verification (echostr).
func (h *WebhookHandler) Verify(c *fiber.Ctx) error {
	echo, err := h.svc.VerifyURL(
		c.Query("signature"),
		c.Query("timestamp"),
		c.Query("nonce"),
		c.Query("echostr"),
		c.Query("token"),
	)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendString(echo)
}

func (h *WebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()
	if err := h.svc.Handle(
		c.Context(),
		c.Query("signature"),
		c.Query("timestamp"),
		c.Query("nonce"),
		body,
		c.Query("token"),
		c.Query("callback_url"),
	); err != nil {
		return utils.BadRequest(c, err)
	}
	return c.SendString("success")
}
