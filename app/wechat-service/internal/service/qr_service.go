package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/laconic-smart-solution/communication-service/app/wechat-service/internal/client"
)

const (
	DefaultQRSize = 256
	minQRSize     = 64
	maxQRSize     = 1024
)

type QRResult struct {
	SceneStr string
	ChatURL  string
	PNG      []byte
	Ticket   string
}

type QRService interface {
	Generate(ctx context.Context, accessToken, sceneStr string, size int) (*QRResult, error)
	GenerateFallbackURL(oaUsername string, size int) (*QRResult, error)
}

type qrServiceImpl struct {
	wechat client.WeChatClient
}

func NewQRService(wechat client.WeChatClient) QRService {
	return &qrServiceImpl{wechat: wechat}
}

func (s *qrServiceImpl) Generate(ctx context.Context, accessToken, sceneStr string, size int) (*QRResult, error) {
	sceneStr = strings.TrimSpace(sceneStr)
	if accessToken == "" {
		return nil, errors.New("X-WeChat-Access-Token header is required for scene QR")
	}
	if sceneStr == "" {
		return nil, errors.New("scene_str query parameter is required")
	}
	if size == 0 {
		size = DefaultQRSize
	}
	if size < minQRSize || size > maxQRSize {
		return nil, fmt.Errorf("size must be between %d and %d", minQRSize, maxQRSize)
	}

	created, err := s.wechat.CreateTempQR(ctx, client.CreateTempQRInput{
		AccessToken: accessToken,
		SceneStr:    sceneStr,
		ExpireSec:   600,
	})
	if err != nil {
		return nil, err
	}

	png, err := s.wechat.FetchQRCodePNG(ctx, created.Ticket)
	if err != nil {
		// Fallback: encode the WeChat QR URL ourselves
		png, err = qrcode.Encode(created.URL, qrcode.Medium, size)
		if err != nil {
			return nil, fmt.Errorf("encode qr: %w", err)
		}
	}

	return &QRResult{
		SceneStr: sceneStr,
		ChatURL:  created.URL,
		PNG:      png,
		Ticket:   created.Ticket,
	}, nil
}

func (s *qrServiceImpl) GenerateFallbackURL(oaUsername string, size int) (*QRResult, error) {
	name := strings.TrimSpace(oaUsername)
	if name == "" {
		return nil, errors.New("oa_username or scene_str required")
	}
	if size == 0 {
		size = DefaultQRSize
	}
	// Profile page deep link (user still sends CONNECT in chat).
	chatURL := "https://mp.weixin.qq.com/mp/profile_ext?action=home&__biz=&scene=124#" + name
	if !strings.HasPrefix(name, "http") {
		chatURL = "weixin://dl/officialaccounts/?username=" + name
	}
	png, err := qrcode.Encode(chatURL, qrcode.Medium, size)
	if err != nil {
		return nil, err
	}
	return &QRResult{SceneStr: "", ChatURL: chatURL, PNG: png}, nil
}

func QRImageDataURI(png []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}
