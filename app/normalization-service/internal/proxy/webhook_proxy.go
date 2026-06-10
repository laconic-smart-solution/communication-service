package proxy

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/laconic-smart-solution/communication-service/app/normalization-service/internal/config"
)

type WebhookProxy struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewWebhookProxy(cfg *config.Config) *WebhookProxy {
	return &WebhookProxy{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *WebhookProxy) Handle(c *fiber.Ctx) error {
	platform := c.Params("platform")
	base := p.cfg.PlatformURL(platform)
	if base == "" {
		return fiber.NewError(fiber.StatusNotFound, "unsupported platform")
	}

	suffix := c.Params("*")
	target := strings.TrimRight(base, "/") + "/" + strings.TrimLeft(suffix, "/")
	if c.Context().QueryArgs().Len() > 0 {
		target += "?" + string(c.Context().QueryArgs().QueryString())
	}

	req, err := http.NewRequestWithContext(c.Context(), c.Method(), target, bytes.NewReader(c.Body()))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "create proxy request")
	}

	c.Request().Header.VisitAll(func(key, value []byte) {
		k := string(key)
		if strings.EqualFold(k, "Host") {
			return
		}
		req.Header.Set(k, string(value))
	})
	if p.cfg.GatewaySharedSecret != "" {
		req.Header.Set("X-Gateway-Secret", p.cfg.GatewaySharedSecret)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "upstream unavailable")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "read upstream response")
	}

	c.Status(resp.StatusCode)
	for k, vals := range resp.Header {
		if len(vals) > 0 {
			c.Set(k, vals[0])
		}
	}
	return c.Send(body)
}
