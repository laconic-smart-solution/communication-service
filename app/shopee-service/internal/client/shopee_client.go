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
	"strconv"
	"strings"
	"time"

	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/config"
)

const (
	authPartnerPath = "/api/v2/shop/auth_partner"
	tokenGetPath    = "/api/v2/auth/token/get"
	tokenRefreshPath = "/api/v2/auth/access_token/get"
)

var (
	ErrInvalidCode  = errors.New("invalid shopee authorization code")
	ErrInvalidToken = errors.New("invalid shopee token")
	ErrRateLimited  = errors.New("shopee api rate limited")
)

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int64
	RefreshExpiresIn int64
	ShopID           string
	Country          string
}

type ShopeeClient interface {
	BuildAuthorizeURL(state string) string
	ExchangeToken(ctx context.Context, code, shopID string) (*TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken, shopID string) (*TokenPair, error)
}

type shopeeClientImpl struct {
	partnerID   string
	partnerKey  string
	redirectURI string
	host        string
	httpClient  *http.Client
}

var _ ShopeeClient = (*shopeeClientImpl)(nil)

func NewShopeeClient(cfg *config.Config, timeoutSec int) ShopeeClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &shopeeClientImpl{
		partnerID:   cfg.ShopeePartnerID,
		partnerKey:  cfg.ShopeePartnerKey,
		redirectURI: cfg.ShopeeRedirectURI,
		host:        strings.TrimRight(cfg.ShopeeHost, "/"),
		httpClient:  &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *shopeeClientImpl) BuildAuthorizeURL(state string) string {
	timestamp := time.Now().Unix()
	sign := signShopee(c.partnerID, authPartnerPath, timestamp, c.partnerKey, "")
	q := url.Values{}
	q.Set("partner_id", c.partnerID)
	q.Set("redirect", c.redirectURI)
	q.Set("timestamp", strconv.FormatInt(timestamp, 10))
	q.Set("sign", sign)
	if state != "" {
		q.Set("state", state)
	}
	return c.host + authPartnerPath + "?" + q.Encode()
}

func (c *shopeeClientImpl) ExchangeToken(ctx context.Context, code, shopID string) (*TokenPair, error) {
	body := map[string]interface{}{
		"code":       code,
		"shop_id":    parseShopID(shopID),
		"partner_id": parsePartnerID(c.partnerID),
	}
	return c.doTokenRequest(ctx, tokenGetPath, body)
}

func (c *shopeeClientImpl) RefreshToken(ctx context.Context, refreshToken, shopID string) (*TokenPair, error) {
	body := map[string]interface{}{
		"refresh_token": refreshToken,
		"shop_id":       parseShopID(shopID),
		"partner_id":    parsePartnerID(c.partnerID),
	}
	return c.doTokenRequest(ctx, tokenRefreshPath, body)
}

func (c *shopeeClientImpl) doTokenRequest(ctx context.Context, apiPath string, body map[string]interface{}) (*TokenPair, error) {
	timestamp := time.Now().Unix()
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}
	sign := signShopee(c.partnerID, apiPath, timestamp, c.partnerKey, string(bodyBytes))

	reqURL := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%d&sign=%s",
		c.host, apiPath, c.partnerID, timestamp, sign)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("shopee api request: %w", err)
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
		Error     string          `json:"error"`
		Message   string          `json:"message"`
		Response  json.RawMessage `json:"response"`
		RequestID string          `json:"request_id"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if envelope.Error != "" {
		if strings.Contains(strings.ToLower(envelope.Error), "invalid") {
			return nil, ErrInvalidCode
		}
		return nil, fmt.Errorf("shopee api error: %s", envelope.Message)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpireIn     int64  `json:"expire_in"`
		ShopID       int64  `json:"shop_id"`
	}
	if err := json.Unmarshal(envelope.Response, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return nil, ErrInvalidCode
	}

	return &TokenPair{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpireIn,
		ShopID:       strconv.FormatInt(tokenResp.ShopID, 10),
		Country:      "SG",
	}, nil
}

func signShopee(partnerID, path string, timestamp int64, partnerKey, body string) string {
	base := fmt.Sprintf("%s%s%d%s", partnerID, path, timestamp, body)
	mac := hmac.New(sha256.New, []byte(partnerKey))
	mac.Write([]byte(base))
	return hex.EncodeToString(mac.Sum(nil))
}

func parseShopID(shopID string) int64 {
	n, _ := strconv.ParseInt(shopID, 10, 64)
	return n
}

func parsePartnerID(partnerID string) int64 {
	n, _ := strconv.ParseInt(partnerID, 10, 64)
	return n
}
