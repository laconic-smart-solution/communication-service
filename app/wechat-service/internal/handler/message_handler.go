package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

const headerAccessToken = "X-WeChat-Access-Token"

type MessageHandler struct {
	svc service.MessageService
}

func NewMessageHandler(svc service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

func (h *MessageHandler) Send(c *fiber.Ctx) error {
	if _, err := utils.RequireTenantID(c); err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}

	accessToken := c.Get(headerAccessToken)
	if accessToken == "" {
		return utils.BadRequest(c, errors.New("X-WeChat-Access-Token header is required"))
	}

	var req request.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}

	res, err := h.svc.Send(c.Context(), accessToken, req)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.Created(c, res)
}

func (h *MessageHandler) SendBatch(c *fiber.Ctx) error {
	if _, err := utils.RequireTenantID(c); err != nil {
		return utils.BadRequest(c, err)
	}
	if _, err := utils.RequireActorID(c); err != nil {
		return utils.Unauthorized(c, err)
	}

	accessToken := c.Get(headerAccessToken)
	if accessToken == "" {
		return utils.BadRequest(c, errors.New("X-WeChat-Access-Token header is required"))
	}

	var req request.SendBatchMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.BadRequest(c, err)
	}

	res, err := h.svc.SendBatch(c.Context(), accessToken, req)
	if err != nil {
		return utils.BadRequest(c, err)
	}
	return utils.Created(c, res)
}
