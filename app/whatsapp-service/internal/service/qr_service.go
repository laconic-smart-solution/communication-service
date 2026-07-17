package service

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/laconic-smart-solution/communication-service/app/whatsapp-service/internal/client"
)

const (
	DefaultQRSize = 256
	minQRSize     = 64
	maxQRSize     = 1024
)

type QRResult struct {
	Phone   string
	ChatURL string
	PNG     []byte
}

type QRService interface {
	Generate(phone, textPayload string, size int) (*QRResult, error)
}

type qrServiceImpl struct{}

var _ QRService = (*qrServiceImpl)(nil)

func NewQRService() QRService {
	return &qrServiceImpl{}
}

func (s *qrServiceImpl) Generate(phone, textPayload string, size int) (*QRResult, error) {
	normalized, err := client.NormalizeBusinessPhone(phone)
	if err != nil {
		return nil, err
	}

	if size == 0 {
		size = DefaultQRSize
	}
	if size < minQRSize || size > maxQRSize {
		return nil, fmt.Errorf("size must be between %d and %d", minQRSize, maxQRSize)
	}

	chatURL, err := client.ChatDeepLink(normalized, textPayload)
	if err != nil {
		return nil, err
	}

	png, err := qrcode.Encode(chatURL, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("encode qr code: %w", err)
	}

	return &QRResult{
		Phone:   normalized,
		ChatURL: chatURL,
		PNG:     png,
	}, nil
}
