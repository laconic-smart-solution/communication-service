package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/handler"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type Handlers struct {
	Store *handler.StoreHandler
}

func Register(app *fiber.App, h Handlers) {
	api := app.Group("/api/v1")
	api.Get("/healthz", func(c *fiber.Ctx) error {
		return utils.Healthz(c, "lazada-service")
	})

	stores := api.Group("/stores")
	stores.Get("/authorize-url", h.Store.AuthorizeURL)
	stores.Post("/connect", h.Store.Connect)
	stores.Get("/", h.Store.List)
	stores.Delete("/:id", h.Store.Revoke)
	stores.Post("/:id/refresh", h.Store.Refresh)
}
