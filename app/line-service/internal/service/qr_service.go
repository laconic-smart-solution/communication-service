package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	DefaultQRSize = 256
	minQRSize     = 64
	maxQRSize     = 1024
)

// LINE basic IDs are an "@" followed by the ID issued to the official account.
var basicIDPattern = regexp.MustCompile(`^@[a-zA-Z0-9._-]{2,20}$`)

type QRResult struct {
	BasicID      string
	AddFriendURL string
	PNG          []byte
}

type QRService interface {
	Generate(basicID string, size int) (*QRResult, error)
}

type qrServiceImpl struct{}

var _ QRService = (*qrServiceImpl)(nil)

func NewQRService() QRService {
	return &qrServiceImpl{}
}

func (s *qrServiceImpl) Generate(basicID string, size int) (*QRResult, error) {
	normalized, err := normalizeBasicID(basicID)
	if err != nil {
		return nil, err
	}

	if size == 0 {
		size = DefaultQRSize
	}
	if size < minQRSize || size > maxQRSize {
		return nil, fmt.Errorf("size must be between %d and %d", minQRSize, maxQRSize)
	}

	// LINE resolves the add-friend page from the basic ID with "@" percent-encoded.
	addFriendURL := "https://line.me/R/ti/p/%40" + strings.TrimPrefix(normalized, "@")

	png, err := qrcode.Encode(addFriendURL, qrcode.Medium, size)
	if err != nil {
		return nil, fmt.Errorf("encode qr code: %w", err)
	}

	return &QRResult{
		BasicID:      normalized,
		AddFriendURL: addFriendURL,
		PNG:          png,
	}, nil
}

func normalizeBasicID(basicID string) (string, error) {
	trimmed := strings.TrimSpace(basicID)
	if trimmed == "" {
		return "", errors.New("basic_id query parameter is required")
	}
	if !strings.HasPrefix(trimmed, "@") {
		trimmed = "@" + trimmed
	}
	if !basicIDPattern.MatchString(trimmed) {
		return "", errors.New("basic_id must look like @136laxkx")
	}
	return trimmed, nil
}
