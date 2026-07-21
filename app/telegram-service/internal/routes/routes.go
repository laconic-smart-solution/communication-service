package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/telegram-service/internal/handler"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type Handlers struct {
	Message *handler.MessageHandler
	Webhook *handler.WebhookHandler
	QR      *handler.QRHandler
}

func Register(app *fiber.App, h Handlers) {
	api := app.Group("/api/v1")
	api.Get("/healthz", func(c *fiber.Ctx) error {
		return utils.Healthz(c, "telegram-service")
	})

	api.Post("/webhooks", h.Webhook.Handle)
	api.Post("/messages", h.Message.Send)
	api.Post("/messages/batch", h.Message.SendBatch)
	api.Get("/qr", h.QR.Get)
}
