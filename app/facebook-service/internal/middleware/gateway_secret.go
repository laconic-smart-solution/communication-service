package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/pkg/response"
)

func RequireGatewaySecret(sharedSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		if path == "/api/v1/healthz" || strings.HasSuffix(path, "/webhooks") {
			return c.Next()
		}
		if sharedSecret == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(response.APIResponse{
				Success: false,
				Error:   "gateway secret is not configured",
			})
		}
		got := c.Get("X-Gateway-Secret")
		if subtle.ConstantTimeCompare([]byte(got), []byte(sharedSecret)) != 1 {
			return c.Status(fiber.StatusForbidden).JSON(response.APIResponse{
				Success: false,
				Error:   "forbidden",
			})
		}
		return c.Next()
	}
}
