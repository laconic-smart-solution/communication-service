package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/handler"
)

type Handlers struct {
	Channel *handler.ChannelHandler
	Message *handler.MessageHandler
	Webhook *handler.WebhookHandler
}

func Register(app *fiber.App, h Handlers) {
	api := app.Group("/api/v1")
	api.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "line-service"})
	})

	api.Post("/webhooks", h.Webhook.Handle)

	channels := api.Group("/channels")
	channels.Post("/", h.Channel.Connect)
	channels.Get("/", h.Channel.List)
	channels.Delete("/:id", h.Channel.Revoke)

	messages := api.Group("/messages")
	messages.Post("/", h.Message.Send)
	messages.Get("/:id", h.Message.GetByID)
}
