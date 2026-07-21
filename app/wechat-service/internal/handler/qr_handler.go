package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/response"
	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/service"
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

	accessToken := c.Get("X-WeChat-Access-Token")
	oaUsername := c.Query("oa_username")
	sceneStr := c.Query("scene_str")
	size := c.QueryInt("size", service.DefaultQRSize)

	var (
		res *service.QRResult
		err error
	)
	if sceneStr != "" {
		res, err = h.svc.Generate(c.Context(), accessToken, sceneStr, size)
	} else {
		res, err = h.svc.GenerateFallbackURL(oaUsername, size)
	}
	if err != nil {
		return utils.BadRequest(c, err)
	}

	if c.Query("format") == "json" {
		return utils.OK(c, response.QRResponse{
			SceneStr: res.SceneStr,
			ChatURL:  res.ChatURL,
			QRImage:  service.QRImageDataURI(res.PNG),
		})
	}

	c.Set(fiber.HeaderContentType, "image/png")
	c.Set(fiber.HeaderCacheControl, "public, max-age=300")
	return c.Send(res.PNG)
}
