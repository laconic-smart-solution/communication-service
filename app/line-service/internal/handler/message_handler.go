package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type MessageHandler struct {
	svc service.MessageService
}

func NewMessageHandler(svc service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

func (h *MessageHandler) Send(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}

	var req request.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}

	res, err := h.svc.Send(c.Context(), tenantID, req, utils.ActorID(c))
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.Created(c, res)
}

func (h *MessageHandler) GetByID(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}

	messageID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, err)
	}

	res, err := h.svc.GetByID(c.Context(), tenantID, messageID)
	if err != nil {
		if err.Error() == "message not found" {
			return utils.NotFound(c, err)
		}
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, res)
}
