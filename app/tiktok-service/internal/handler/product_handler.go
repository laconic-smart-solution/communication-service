package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) List(c *fiber.Ctx) error {
	creds, err := requireTikTokCredentials(c)
	if err != nil {
		return badRequestCredentials(c, err)
	}

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

	res, err := h.svc.List(c.Context(), creds, req)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, res)
}

func (h *ProductHandler) Get(c *fiber.Ctx) error {
	creds, err := requireTikTokCredentials(c)
	if err != nil {
		return badRequestCredentials(c, err)
	}

	if _, err := utils.RequireTenantID(c); err != nil {
		return utils.BadRequest(c, err)
	}

	var req request.GetProductRequest
	if err := c.ParamsParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := c.QueryParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}

	res, err := h.svc.Get(c.Context(), creds, req)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return utils.NotFound(c, err)
		}
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, res)
}
