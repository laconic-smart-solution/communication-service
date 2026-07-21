package handler

import (
	"encoding/base64"

	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/viber-service/internal/response"
	"github.com/laconic-smart-solution/communication-service/app/viber-service/internal/service"
	"github.com/laconic-smart-solution/communication-service/pkg/utils"
)

type QRHandler struct {
	svc service.QRService
}

func NewQRHandler(svc service.QRService) *QRHandler {
	return &QRHandler{svc: svc}
}

func (h *QRHandler) Get(c *fiber.Ctx) error {
	if _, err := utils.RequireTenantID(c); err != nil {
		return utils.BadRequest(c, err)
	}

	res, err := h.svc.Generate(
		c.Query("uri"),
		c.Query("start"),
		c.QueryInt("size", service.DefaultQRSize),
	)
	if err != nil {
		return utils.BadRequest(c, err)
	}

	if c.Query("format") == "json" {
		return utils.OK(c, response.QRResponse{
			URI: res.URI,
			BotURL:      res.BotURL,
			QRImage:     "data:image/png;base64," + base64.StdEncoding.EncodeToString(res.PNG),
		})
	}

	c.Set(fiber.HeaderContentType, "image/png")
	c.Set(fiber.HeaderCacheControl, "public, max-age=86400")
	return c.Send(res.PNG)
}
