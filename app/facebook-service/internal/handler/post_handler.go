package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type PostHandler struct {
	svc service.PostService
}

func NewPostHandler(svc service.PostService) *PostHandler {
	return &PostHandler{svc: svc}
}

func (h *PostHandler) Create(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}
	var req request.CreatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}
	res, err := h.svc.Create(c.Context(), tenantID, req, utils.ActorID(c))
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.Created(c, res)
}

func (h *PostHandler) List(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	var req request.ListPostsRequest
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

func (h *PostHandler) GetByID(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, err)
	}
	res, err := h.svc.GetByID(c.Context(), tenantID, postID)
	if err != nil {
		if err.Error() == "post not found" {
			return utils.NotFound(c, err)
		}
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, res)
}

func (h *PostHandler) Update(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}
	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, err)
	}
	var req request.UpdatePostRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}
	res, err := h.svc.Update(c.Context(), tenantID, postID, req, utils.ActorID(c))
	if err != nil {
		if err.Error() == "post not found" {
			return utils.NotFound(c, err)
		}
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, res)
}

func (h *PostHandler) Delete(c *fiber.Ctx) error {
	tenantID, err := utils.RequireTenantID(c)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}
	postID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, err)
	}
	if err := h.svc.Delete(c.Context(), tenantID, postID, utils.ActorID(c)); err != nil {
		if err.Error() == "post not found" {
			return utils.NotFound(c, err)
		}
		return utils.BadRequest(c, err)
	}
	return utils.OK(c, fiber.Map{"deleted": true})
}
