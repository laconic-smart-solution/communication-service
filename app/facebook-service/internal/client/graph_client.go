package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrRateLimited      = errors.New("facebook api rate limited")
	ErrInvalidToken     = errors.New("invalid facebook access token")
)

// GraphClient wraps Meta Graph API for Page feed posts.
// See https://developers.facebook.com/docs/graph-api/
type GraphClient interface {
	CreatePost(ctx context.Context, pageID, accessToken, message string) (externalID string, err error)
	GetPost(ctx context.Context, postID, accessToken string) (message string, err error)
	UpdatePost(ctx context.Context, postID, accessToken, message string) error
	DeletePost(ctx context.Context, postID, accessToken string) error
	VerifyWebhook(signature string, body []byte, appSecret string) error
}

type graphClientImpl struct {
	apiVersion string
	httpClient *http.Client
}

var _ GraphClient = (*graphClientImpl)(nil)

func NewGraphClient(apiVersion string, timeoutSec int) GraphClient {
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &graphClientImpl{
		apiVersion: apiVersion,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *graphClientImpl) baseURL() string {
	return "https://graph.facebook.com/" + c.apiVersion
}

func (c *graphClientImpl) CreatePost(ctx context.Context, pageID, accessToken, message string) (string, error) {
	form := url.Values{}
	form.Set("message", message)
	form.Set("access_token", accessToken)

	var resp struct {
		ID    string `json:"id"`
		Error *fbError `json:"error"`
	}
	if err := c.postForm(ctx, c.baseURL()+"/"+pageID+"/feed", form, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", mapFBError(resp.Error)
	}
	if resp.ID == "" {
		return "", errors.New("facebook api returned empty post id")
	}
	return resp.ID, nil
}

func (c *graphClientImpl) GetPost(ctx context.Context, postID, accessToken string) (string, error) {
	u := fmt.Sprintf("%s/%s?fields=id,message&access_token=%s", c.baseURL(), postID, url.QueryEscape(accessToken))
	var resp struct {
		Message string `json:"message"`
		Error   *fbError `json:"error"`
	}
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", mapFBError(resp.Error)
	}
	return resp.Message, nil
}

func (c *graphClientImpl) UpdatePost(ctx context.Context, postID, accessToken, message string) error {
	form := url.Values{}
	form.Set("message", message)
	form.Set("access_token", accessToken)

	var resp struct {
		Success bool `json:"success"`
		Error   *fbError `json:"error"`
	}
	if err := c.postForm(ctx, c.baseURL()+"/"+postID, form, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return mapFBError(resp.Error)
	}
	return nil
}

func (c *graphClientImpl) DeletePost(ctx context.Context, postID, accessToken string) error {
	u := fmt.Sprintf("%s/%s?access_token=%s", c.baseURL(), postID, url.QueryEscape(accessToken))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body struct {
		Success bool `json:"success"`
		Error   *fbError `json:"error"`
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	_ = json.Unmarshal(raw, &body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	if body.Error != nil {
		return mapFBError(body.Error)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("facebook api status %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (c *graphClientImpl) VerifyWebhook(signature string, body []byte, appSecret string) error {
	if appSecret == "" {
		return errors.New("facebook app secret is not configured")
	}
	if signature == "" || !strings.HasPrefix(signature, "sha256=") {
		return ErrInvalidSignature
	}
	expected := signature[7:]
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	actual := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(actual), []byte(expected)) {
		return ErrInvalidSignature
	}
	return nil
}

type fbError struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Code      int    `json:"code"`
	ErrorCode int    `json:"error_subcode"`
}

func mapFBError(e *fbError) error {
	if e == nil {
		return errors.New("facebook api error")
	}
	if e.Code == 190 || e.Type == "OAuthException" {
		return ErrInvalidToken
	}
	if e.Code == 4 || e.Code == 17 {
		return ErrRateLimited
	}
	return fmt.Errorf("facebook api: %s", e.Message)
}

func (c *graphClientImpl) postForm(ctx context.Context, endpoint string, form url.Values, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req, out)
}

func (c *graphClientImpl) getJSON(ctx context.Context, endpoint string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *graphClientImpl) do(req *http.Request, out interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode facebook response: %w", err)
	}
	return nil
}
