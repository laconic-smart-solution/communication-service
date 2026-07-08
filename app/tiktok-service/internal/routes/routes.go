package routes

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/handler"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type Handlers struct {
	Store *handler.StoreHandler
	Order *handler.OrderHandler
}

func Register(app *fiber.App, h Handlers) {
	api := app.Group("/api/v1")
	api.Get("/healthz", func(c *fiber.Ctx) error {
		return utils.Healthz(c, "tiktok-service")
	})

	if h.Store != nil {
		stores := api.Group("/stores")
		stores.Get("/authorize-url", h.Store.AuthorizeURL)
		stores.Post("/connect", h.Store.Connect)
		stores.Post("/refresh", h.Store.Refresh)
	}

	if h.Order != nil {
		orders := api.Group("/orders")
		orders.Get("/", h.Order.List)
	}
}
