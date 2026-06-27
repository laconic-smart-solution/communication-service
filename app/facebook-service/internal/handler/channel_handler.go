package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type ChannelHandler struct {
	svc service.ChannelService
}

func NewChannelHandler(svc service.ChannelService) *ChannelHandler {
	return &ChannelHandler{svc: svc}
}

func (h *ChannelHandler) Connect(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}
	var req request.ConnectChannelRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}
	res, err := h.svc.Connect(c.Context(), tenantID, req, utils.ActorID(c))
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.Created(c, res)
}

func (h *ChannelHandler) List(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	var req request.ListChannelsRequest
	if err := c.QueryParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}
	res, err := h.svc.List(c.Context(), tenantID, req)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, res)
}

func (h *ChannelHandler) Revoke(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if err := h.svc.Revoke(c.Context(), tenantID, channelID, utils.ActorID(c)); err != nil {
		if err.Error() == "channel not found" {
			return utils.NotFound(c, err)
		}
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, fiber.Map{"revoked": true})
}
