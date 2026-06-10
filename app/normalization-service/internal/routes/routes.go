package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/normalization-service/internal/handler"
	"github.com/laconic-smart-solution/communication-service/app/normalization-service/internal/proxy"
)

type Handlers struct {
	GraphQL *handler.GraphQLHandler
	Proxy   *proxy.WebhookProxy
}

func Register(app *fiber.App, h Handlers, playground bool) {
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "normalization-service"})
	})

	app.Post("/graphql", h.GraphQL.Handle)
	if playground {
		app.Get("/graphql", h.GraphQL.Playground)
	}

	app.All("/proxy/:platform/*", h.Proxy.Handle)
}
