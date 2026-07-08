package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type OrderHandler struct {
	svc service.OrderService
}

func NewOrderHandler(svc service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) List(c *fiber.Ctx) error {
	if _, err := utils.RequireTenantID(c); err != nil {
		return utils.BadRequest(c, err)
	}

	var req request.ListOrdersRequest
	if err := c.QueryParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}

	items, err := h.svc.List(c.Context(), req.AccessToken, req.SellerID, req.UpdatedFrom)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, items)
}
