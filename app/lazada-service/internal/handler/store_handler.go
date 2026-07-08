package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type StoreHandler struct {
	svc service.StoreService
}

func NewStoreHandler(svc service.StoreService) *StoreHandler {
	return &StoreHandler{svc: svc}
}

func (h *StoreHandler) AuthorizeURL(c *fiber.Ctx) error {
	var req request.AuthorizeURLRequest
	if err := c.QueryParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, h.svc.AuthorizeURL(req.State))
}

func (h *StoreHandler) Connect(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}

	var req request.ConnectStoreRequest
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

func (h *StoreHandler) Refresh(c *fiber.Ctx) error {
	if _, err := utils.RequireTenantID(c); err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}

	var req request.RefreshStoreRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}

	res, err := h.svc.Refresh(c.Context(), req)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, res)
}
