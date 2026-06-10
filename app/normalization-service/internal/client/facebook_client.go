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

type PostDTO struct {
	ID         uuid.UUID `json:"id"`
	ChannelID  uuid.UUID `json:"channel_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Message    string    `json:"message"`
	Status     string    `json:"status"`
	ExternalID string    `json:"external_id"`
}

type PagedPosts struct {
	Items []PostDTO `json:"items"`
}

type FacebookClient interface {
	ListPosts(ctx context.Context, headers map[string]string, channelID string, page, limit int) ([]PostDTO, error)
	GetPost(ctx context.Context, headers map[string]string, id uuid.UUID) (*PostDTO, error)
	CreatePost(ctx context.Context, headers map[string]string, channelID uuid.UUID, message string) (*PostDTO, error)
	UpdatePost(ctx context.Context, headers map[string]string, id uuid.UUID, message string) (*PostDTO, error)
	DeletePost(ctx context.Context, headers map[string]string, id uuid.UUID) error
	ConnectChannel(ctx context.Context, headers map[string]string, externalID, displayName, accessToken string) (*ChannelDTO, error)
	ListChannels(ctx context.Context, headers map[string]string, page, limit int) ([]ChannelDTO, error)
	RevokeChannel(ctx context.Context, headers map[string]string, id uuid.UUID) error
}

type facebookClientImpl struct {
	baseURL    string
	httpClient *http.Client
}

var _ FacebookClient = (*facebookClientImpl)(nil)

func NewFacebookClient(baseURL string, timeoutSec int) FacebookClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &facebookClientImpl{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *facebookClientImpl) ListPosts(ctx context.Context, headers map[string]string, channelID string, page, limit int) ([]PostDTO, error) {
	url := fmt.Sprintf("%s/api/v1/posts?page=%d&limit=%d", c.baseURL, page, limit)
	if channelID != "" {
		url += "&channel_id=" + channelID
	}
	var env APIEnvelope
	if err := c.do(ctx, http.MethodGet, url, headers, nil, &env); err != nil {
		return nil, err
	}
	if !env.Success {
		return nil, fmt.Errorf(env.Error)
	}
	var pageData PagedPosts
	if err := json.Unmarshal(env.Data, &pageData); err != nil {
		return nil, err
	}
	return pageData.Items, nil
}

func (c *facebookClientImpl) GetPost(ctx context.Context, headers map[string]string, id uuid.UUID) (*PostDTO, error) {
	var env APIEnvelope
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/posts/%s", c.baseURL, id), headers, nil, &env); err != nil {
		return nil, err
	}
	if !env.Success {
		return nil, fmt.Errorf(env.Error)
	}
	var post PostDTO
	if err := json.Unmarshal(env.Data, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (c *facebookClientImpl) CreatePost(ctx context.Context, headers map[string]string, channelID uuid.UUID, message string) (*PostDTO, error) {
	body, _ := json.Marshal(map[string]interface{}{"channel_id": channelID, "message": message})
	var env APIEnvelope
	if err := c.do(ctx, http.MethodPost, c.baseURL+"/api/v1/posts", headers, body, &env); err != nil {
		return nil, err
	}
	if !env.Success {
		return nil, fmt.Errorf(env.Error)
	}
	var post PostDTO
	if err := json.Unmarshal(env.Data, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (c *facebookClientImpl) UpdatePost(ctx context.Context, headers map[string]string, id uuid.UUID, message string) (*PostDTO, error) {
	body, _ := json.Marshal(map[string]string{"message": message})
	var env APIEnvelope
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("%s/api/v1/posts/%s", c.baseURL, id), headers, body, &env); err != nil {
		return nil, err
	}
	if !env.Success {
		return nil, fmt.Errorf(env.Error)
	}
	var post PostDTO
	if err := json.Unmarshal(env.Data, &post); err != nil {
		return nil, err
	}
	return &post, nil
}

func (c *facebookClientImpl) DeletePost(ctx context.Context, headers map[string]string, id uuid.UUID) error {
	var env APIEnvelope
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("%s/api/v1/posts/%s", c.baseURL, id), headers, nil, &env); err != nil {
		return err
	}
	if !env.Success {
		return fmt.Errorf(env.Error)
	}
	return nil
}

func (c *facebookClientImpl) ConnectChannel(ctx context.Context, headers map[string]string, externalID, displayName, accessToken string) (*ChannelDTO, error) {
	body, _ := json.Marshal(map[string]string{
		"external_id": externalID, "display_name": displayName, "access_token": accessToken,
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

func (c *facebookClientImpl) ListChannels(ctx context.Context, headers map[string]string, page, limit int) ([]ChannelDTO, error) {
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

func (c *facebookClientImpl) RevokeChannel(ctx context.Context, headers map[string]string, id uuid.UUID) error {
	var env APIEnvelope
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("%s/api/v1/channels/%s", c.baseURL, id), headers, nil, &env); err != nil {
		return err
	}
	if !env.Success {
		return fmt.Errorf(env.Error)
	}
	return nil
}

func (c *facebookClientImpl) do(ctx context.Context, method, url string, headers map[string]string, body []byte, out *APIEnvelope) error {
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
	return json.Unmarshal(raw, out)
}
