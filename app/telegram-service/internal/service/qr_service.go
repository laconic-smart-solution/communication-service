package service

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/laconic-smart-solution/communication-service/app/telegram-service/internal/client"
)

const (
	DefaultQRSize = 256
	minQRSize     = 64
	maxQRSize     = 1024
)

type QRResult struct {
	BotUsername string
	BotURL      string
	PNG         []byte
}

type QRService interface {
	Generate(botUsername, startPayload string, size int) (*QRResult, error)
}

type qrServiceImpl struct{}

var _ QRService = (*qrServiceImpl)(nil)

func NewQRService() QRService {
	return &qrServiceImpl{}
}

func (s *qrServiceImpl) Generate(botUsername, startPayload string, size int) (*QRResult, error) {
	normalized, err := client.NormalizeBotUsername(botUsername)
	if err != nil {
		return nil, err
	}

	if size == 0 {
		size = DefaultQRSize
	}
	if size < minQRSize || size > maxQRSize {
		return nil, fmt.Errorf("size must be between %d and %d", minQRSize, maxQRSize)
	}

	botURL, err := client.BotDeepLink(normalized, startPayload)
	if err != nil {
		return nil, err
	}

	png, err := qrcode.Encode(botURL, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("encode qr code: %w", err)
	}

	return &QRResult{
		BotUsername: normalized,
		BotURL:      botURL,
		PNG:         png,
	}, nil
}
