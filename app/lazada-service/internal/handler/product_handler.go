package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) List(c *fiber.Ctx) error {
	if _, err := utils.RequireTenantID(c); err != nil {
		return utils.BadRequest(c, err)
	}

	var req request.ListProductsRequest
	if err := c.QueryParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}

	res, err := h.svc.List(c.Context(), req)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, res)
}
