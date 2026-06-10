package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type APIEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

type ChannelDTO struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	ExternalID  string    `json:"external_id"`
	DisplayName string    `json:"display_name"`
}

type MessageDTO struct {
	ID          uuid.UUID `json:"id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	RecipientID string    `json:"recipient_id"`
	Body        string    `json:"body"`
	Status      string    `json:"status"`
}

type PagedChannels struct {
	Items []ChannelDTO `json:"items"`
}

type LineClient interface {
	ListChannels(ctx context.Context, headers map[string]string, page, limit int) ([]ChannelDTO, error)
	SendMessage(ctx context.Context, headers map[string]string, channelID uuid.UUID, recipientID, text string) (*MessageDTO, error)
	GetMessage(ctx context.Context, headers map[string]string, id uuid.UUID) (*MessageDTO, error)
	ConnectChannel(ctx context.Context, headers map[string]string, externalID, displayName, accessToken string) (*ChannelDTO, error)
	RevokeChannel(ctx context.Context, headers map[string]string, id uuid.UUID) error
}

type lineClientImpl struct {
	baseURL    string
	httpClient *http.Client
}

var _ LineClient = (*lineClientImpl)(nil)

func NewLineClient(baseURL string, timeoutSec int) LineClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &lineClientImpl{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *lineClientImpl) ListChannels(ctx context.Context, headers map[string]string, page, limit int) ([]ChannelDTO, error) {
	url := fmt.Sprintf("%s/api/v1/channels?page=%d&limit=%d", c.baseURL, page, limit)
	var env APIEnvelope
	if err := c.do(ctx, http.MethodGet, url, headers, nil, &env); err != nil {
		return nil, err
	}
	if !env.Success {
		return nil, fmt.Errorf(env.Error)
	}
	var pageData PagedChannels
	if err := json.Unmarshal(env.Data, &pageData); err != nil {
		return nil, err
	}
	return pageData.Items, nil
}

func (c *lineClientImpl) SendMessage(ctx context.Context, headers map[string]string, channelID uuid.UUID, recipientID, text string) (*MessageDTO, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"channel_id":   channelID,
		"recipient_id": recipientID,
		"text":         text,
	})
	var env APIEnvelope
	if err := c.do(ctx, http.MethodPost, c.baseURL+"/api/v1/messages", headers, body, &env); err != nil {
		return nil, err
	}
	if !env.Success {
		return nil, fmt.Errorf(env.Error)
	}
	var msg MessageDTO
	if err := json.Unmarshal(env.Data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *lineClientImpl) GetMessage(ctx context.Context, headers map[string]string, id uuid.UUID) (*MessageDTO, error) {
	var env APIEnvelope
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/messages/%s", c.baseURL, id), headers, nil, &env); err != nil {
		return nil, err
	}
	if !env.Success {
		return nil, fmt.Errorf(env.Error)
	}
	var msg MessageDTO
	if err := json.Unmarshal(env.Data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *lineClientImpl) ConnectChannel(ctx context.Context, headers map[string]string, externalID, displayName, accessToken string) (*ChannelDTO, error) {
	body, _ := json.Marshal(map[string]string{
		"external_id":  externalID,
		"display_name": displayName,
		"access_token": accessToken,
	})
	var env APIEnvelope
	if err := c.do(ctx, http.MethodPost, c.baseURL+"/api/v1/channels", headers, body, &env); err != nil {
		return nil, err
	}
	if !env.Success {
		return nil, fmt.Errorf(env.Error)
	}
	var ch ChannelDTO
	if err := json.Unmarshal(env.Data, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

func (c *lineClientImpl) RevokeChannel(ctx context.Context, headers map[string]string, id uuid.UUID) error {
	var env APIEnvelope
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("%s/api/v1/channels/%s", c.baseURL, id), headers, nil, &env); err != nil {
		return err
	}
	if !env.Success {
		return fmt.Errorf(env.Error)
	}
	return nil
}

func (c *lineClientImpl) do(ctx context.Context, method, url string, headers map[string]string, body []byte, out *APIEnvelope) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
