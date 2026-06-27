package utils

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/laconic-smart-solution/communication-service/pkg/response"
)

func OK(c *fiber.Ctx, data interface{}, message ...string) error {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	return c.Status(fiber.StatusOK).JSON(response.APIResponse{Success: true, Message: msg, Data: data})
}

func Created(c *fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(response.APIResponse{Success: true, Data: data})
}

func Error(c *fiber.Ctx, status int, err error) error {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return c.Status(status).JSON(response.APIResponse{Success: false, Error: msg})
}

func BadRequest(c *fiber.Ctx, err error) error   { return Error(c, fiber.StatusBadRequest, err) }
func NotFound(c *fiber.Ctx, err error) error     { return Error(c, fiber.StatusNotFound, err) }
func Unauthorized(c *fiber.Ctx, err error) error { return Error(c, fiber.StatusUnauthorized, err) }
func Internal(c *fiber.Ctx, err error) error     { return Error(c, fiber.StatusInternalServerError, err) }

func Healthz(c *fiber.Ctx, service string) error {
	return OK(c, response.HealthData{Status: "ok", Service: service})
}

func RequireTenantID(c *fiber.Ctx) (uuid.UUID, error) {
	raw := c.Get("X-Tenant-ID")
	if raw == "" {
		raw = "00000000-0000-0000-0000-000000000001"
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "invalid tenant identity")
	}
	return id, nil
}

func RequireActorID(c *fiber.Ctx) (uuid.UUID, error) {
	raw := c.Get("X-Actor-ID")
	if raw == "" {
		raw = c.Get("X-User-ID")
	}
	if raw == "" {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "missing actor identity")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "invalid actor identity")
	}
	return id, nil
}

func ActorID(c *fiber.Ctx) *uuid.UUID {
	id, err := RequireActorID(c)
	if err != nil {
		return nil
	}
	return &id
}
