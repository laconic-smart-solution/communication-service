package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/config"
)

const tokenGetPath = "/api/v2/token/get"

var (
	ErrInvalidCode  = errors.New("invalid tiktok authorization code")
	ErrInvalidToken = errors.New("invalid tiktok token")
	ErrRateLimited  = errors.New("tiktok api rate limited")
)

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int64
	RefreshExpiresIn int64
	ShopID           string
	Country          string
	Account          string
	ShortCode        string
}

type TikTokClient interface {
	BuildAuthorizeURL(state string) string
	ExchangeToken(ctx context.Context, code string) (*TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
}

type tiktokClientImpl struct {
	appKey      string
	appSecret   string
	redirectURI string
	authHost    string
	httpClient  *http.Client
}

var _ TikTokClient = (*tiktokClientImpl)(nil)

func NewTikTokClient(cfg *config.Config, timeoutSec int) TikTokClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &tiktokClientImpl{
		appKey:      cfg.TikTokAppKey,
		appSecret:   cfg.TikTokAppSecret,
		redirectURI: cfg.TikTokRedirectURI,
		authHost:    strings.TrimRight(cfg.TikTokAuthHost, "/"),
		httpClient:  &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *tiktokClientImpl) BuildAuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("app_key", c.appKey)
	q.Set("state", state)
	return "https://services.tiktokshop.com/open/authorize?" + q.Encode()
}

func (c *tiktokClientImpl) ExchangeToken(ctx context.Context, code string) (*TokenPair, error) {
	body := map[string]string{
		"app_key":    c.appKey,
		"app_secret": c.appSecret,
		"auth_code":  code,
		"grant_type": "authorized_code",
	}
	return c.doTokenRequest(ctx, body)
}

func (c *tiktokClientImpl) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	body := map[string]string{
		"app_key":       c.appKey,
		"app_secret":    c.appSecret,
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}
	return c.doTokenRequest(ctx, body)
}

func (c *tiktokClientImpl) doTokenRequest(ctx context.Context, body map[string]string) (*TokenPair, error) {
	form := url.Values{}
	for k, v := range body {
		form.Set(k, v)
	}

	reqURL := c.authHost + tokenGetPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tiktok api request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}

	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if envelope.Code != 0 {
		if strings.Contains(strings.ToLower(envelope.Message), "invalid") {
			return nil, ErrInvalidCode
		}
		return nil, fmt.Errorf("tiktok api error: %s", envelope.Message)
	}

	var tokenResp struct {
		AccessToken          string `json:"access_token"`
		RefreshToken         string `json:"refresh_token"`
		AccessTokenExpireIn  int64  `json:"access_token_expire_in"`
		RefreshTokenExpireIn int64  `json:"refresh_token_expire_in"`
		OpenID               string `json:"open_id"`
		SellerName           string `json:"seller_name"`
		SellerBaseRegion     string `json:"seller_base_region"`
	}
	if err := json.Unmarshal(envelope.Data, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode token data: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return nil, ErrInvalidCode
	}

	shopID := tokenResp.OpenID
	display := tokenResp.SellerName
	if display == "" {
		display = shopID
	}

	return &TokenPair{
		AccessToken:      tokenResp.AccessToken,
		RefreshToken:     tokenResp.RefreshToken,
		ExpiresIn:        tokenResp.AccessTokenExpireIn,
		RefreshExpiresIn: tokenResp.RefreshTokenExpireIn,
		ShopID:           shopID,
		Country:          tokenResp.SellerBaseRegion,
		Account:          display,
		ShortCode:        shopID,
	}, nil
}
