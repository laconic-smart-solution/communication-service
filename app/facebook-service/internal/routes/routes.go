package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/handler"
)

type Handlers struct {
	Channel *handler.ChannelHandler
	Post    *handler.PostHandler
	Webhook *handler.WebhookHandler
}

func Register(app *fiber.App, h Handlers) {
	api := app.Group("/api/v1")
	api.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "facebook-service"})
	})

	api.Get("/webhooks", h.Webhook.Verify)
	api.Post("/webhooks", h.Webhook.Handle)

	channels := api.Group("/channels")
	channels.Post("/", h.Channel.Connect)
	channels.Get("/", h.Channel.List)
	channels.Delete("/:id", h.Channel.Revoke)

	posts := api.Group("/posts")
	posts.Post("/", h.Post.Create)
	posts.Get("/", h.Post.List)
	posts.Get("/:id", h.Post.GetByID)
	posts.Put("/:id", h.Post.Update)
	posts.Delete("/:id", h.Post.Delete)
}
