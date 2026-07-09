package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/config"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
)

const (
	authBaseURL      = "https://auth.lazada.com/rest"
	defaultAuthURL   = "https://auth.lazada.com/oauth/authorize"
	tokenCreatePath  = "/auth/token/create"
	tokenRefreshPath = "/auth/token/refresh"
)

var (
	ErrInvalidCode  = errors.New("invalid lazada authorization code")
	ErrInvalidToken = errors.New("invalid lazada token")
	ErrRateLimited  = errors.New("lazada api rate limited")
)

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int64
	RefreshExpiresIn int64
	Account          string
	Country          string
	SellerID         string
	ShortCode        string
}

type LazadaClient interface {
	BuildAuthorizeURL(state string) string
	ExchangeToken(ctx context.Context, code string) (*TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	ListProducts(ctx context.Context, accessToken string, page, limit int, search string) (*response.ProductListResponse, error)
}

type lazadaClientImpl struct {
	appKey      string
	appSecret   string
	redirectURI string
	country     string
	forceAuth   bool
	authBaseURL string
	httpClient  *http.Client
}

var _ LazadaClient = (*lazadaClientImpl)(nil)

func NewLazadaClient(cfg *config.Config, timeoutSec int) LazadaClient {
	if timeoutSec < 1 {
		timeoutSec = 15
	}
	return &lazadaClientImpl{
		appKey:      cfg.LazadaAppKey,
		appSecret:   cfg.LazadaAppSecret,
		redirectURI: cfg.LazadaRedirectURI,
		country:     cfg.LazadaCountry,
		forceAuth:   cfg.LazadaForceAuth,
		authBaseURL: resolveAuthorizeBaseURL(cfg.LazadaAuthBaseURL, cfg.LazadaCountry),
		httpClient:  &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func (c *lazadaClientImpl) BuildAuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.appKey)
	q.Set("redirect_uri", c.redirectURI)
	if c.forceAuth {
		q.Set("force_auth", "true")
	}
	if c.country != "" {
		q.Set("country", c.country)
	}
	if state != "" {
		q.Set("state", state)
	}
	return c.authBaseURL + "?" + q.Encode()
}

func resolveAuthorizeBaseURL(explicitBaseURL, country string) string {
	if explicitBaseURL != "" {
		return explicitBaseURL
	}
	if regional, ok := regionalAuthorizeURL(country); ok {
		return regional
	}
	return defaultAuthURL
}

func regionalAuthorizeURL(country string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(country)) {
	case "th":
		return "https://api.lazada.co.th/oauth/authorize", true
	case "sg":
		return "https://api.lazada.sg/oauth/authorize", true
	case "my":
		return "https://api.lazada.com.my/oauth/authorize", true
	case "ph":
		return "https://api.lazada.com.ph/oauth/authorize", true
	case "id":
		return "https://api.lazada.co.id/oauth/authorize", true
	case "vn":
		return "https://api.lazada.vn/oauth/authorize", true
	default:
		return "", false
	}
}

func (c *lazadaClientImpl) ExchangeToken(ctx context.Context, code string) (*TokenPair, error) {
	params := map[string]string{
		"app_key":     c.appKey,
		"code":        code,
		"sign_method": "sha256",
		"timestamp":   strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	params["sign"] = sign(tokenCreatePath, params, c.appSecret)
	return c.doTokenRequest(ctx, tokenCreatePath, params)
}

func (c *lazadaClientImpl) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	params := map[string]string{
		"app_key":       c.appKey,
		"refresh_token": refreshToken,
		"sign_method":   "sha256",
		"timestamp":     strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	params["sign"] = sign(tokenRefreshPath, params, c.appSecret)
	return c.doTokenRequest(ctx, tokenRefreshPath, params)
}

func (c *lazadaClientImpl) doTokenRequest(ctx context.Context, apiPath string, params map[string]string) (*TokenPair, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	reqURL := authBaseURL + apiPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lazada api request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}

	var raw tokenResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if raw.Code != "" && raw.Code != "0" {
		if raw.Code == "InvalidCode" {
			return nil, ErrInvalidCode
		}
		if raw.Code == "IllegalRefreshToken" {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("lazada api error: %s", raw.Message)
	}

	pair, err := raw.toTokenPair()
	if err != nil {
		return nil, err
	}
	if pair.AccessToken == "" {
		return nil, fmt.Errorf("lazada api: empty access token")
	}
	return pair, nil
}

type tokenResponse struct {
	Code             string          `json:"code"`
	Message          string          `json:"message"`
	AccessToken      string          `json:"access_token"`
	RefreshToken     string          `json:"refresh_token"`
	ExpiresIn        json.Number     `json:"expires_in"`
	RefreshExpiresIn json.Number     `json:"refresh_expires_in"`
	Account          string          `json:"account"`
	Country          string          `json:"country"`
	CountryUserInfo  json.RawMessage `json:"country_user_info"`
	CountryUserList  json.RawMessage `json:"country_user_info_list"`
}

type countryUserEntry struct {
	SellerID  string `json:"seller_id"`
	UserID    string `json:"user_id"`
	ShortCode string `json:"short_code"`
}

func (r *tokenResponse) toTokenPair() (*TokenPair, error) {
	expiresIn, _ := r.ExpiresIn.Int64()
	refreshExpiresIn, _ := r.RefreshExpiresIn.Int64()

	pair := &TokenPair{
		AccessToken:      r.AccessToken,
		RefreshToken:     r.RefreshToken,
		ExpiresIn:        expiresIn,
		RefreshExpiresIn: refreshExpiresIn,
		Account:          r.Account,
		Country:          r.Country,
	}

	infoRaw := r.CountryUserInfo
	if len(infoRaw) == 0 {
		infoRaw = r.CountryUserList
	}
	if len(infoRaw) > 0 {
		sellerID, shortCode := parseCountryUserInfo(infoRaw, r.Country)
		pair.SellerID = sellerID
		pair.ShortCode = shortCode
	}

	return pair, nil
}

func parseCountryUserInfo(raw json.RawMessage, country string) (sellerID, shortCode string) {
	var byCountry map[string]countryUserEntry
	if err := json.Unmarshal(raw, &byCountry); err == nil {
		if entry, ok := byCountry[country]; ok {
			return entry.SellerID, entry.ShortCode
		}
		for _, entry := range byCountry {
			if entry.SellerID != "" {
				return entry.SellerID, entry.ShortCode
			}
		}
	}

	var list []struct {
		Country string `json:"country"`
		countryUserEntry
	}
	if err := json.Unmarshal(raw, &list); err == nil {
		for _, entry := range list {
			if entry.Country == country || country == "" {
				return entry.SellerID, entry.ShortCode
			}
		}
		if len(list) > 0 {
			return list[0].SellerID, list[0].ShortCode
		}
	}

	return "", ""
}
