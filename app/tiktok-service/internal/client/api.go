package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const apiVersion = "202309"

type apiEnvelope struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	RequestID string          `json:"request_id"`
	Data      json.RawMessage `json:"data"`
}

func (c *tiktokClientImpl) doSignedJSON(
	ctx context.Context,
	creds Credentials,
	method, path, accessToken, shopCipher string,
	query map[string]string,
	body any,
) (json.RawMessage, error) {
	if query == nil {
		query = map[string]string{}
	}
	query["app_key"] = creds.AppKey
	query["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)

	// shop_cipher is not allowed on authorization/seller endpoints
	if shopCipher != "" && !strings.HasPrefix(path, "/authorization/") && !strings.HasPrefix(path, "/seller/") {
		query["shop_cipher"] = shopCipher
	}

	var bodyBytes []byte
	var bodyStr string
	if body != nil && method != http.MethodGet {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyStr = string(bodyBytes)
	}

	query["sign"] = signRequest(path, query, bodyStr, creds.AppSecret)

	q := url.Values{}
	for k, v := range query {
		q.Set(k, v)
	}
	reqURL := c.apiHost + path + "?" + q.Encode()

	var bodyReader io.Reader
	if len(bodyBytes) > 0 {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if accessToken != "" {
		req.Header.Set("x-tts-access-token", accessToken)
	}
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}

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

	var envelope apiEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if envelope.Code != 0 {
		msg := strings.ToLower(envelope.Message)
		if strings.Contains(msg, "access token") || strings.Contains(msg, "invalid token") {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("tiktok api error (%d): %s", envelope.Code, envelope.Message)
	}
	return envelope.Data, nil
}

func currencyForRegion(region string) string {
	switch strings.ToUpper(strings.TrimSpace(region)) {
	case "TH", "THA":
		return "THB"
	case "MY", "MYS":
		return "MYR"
	case "SG", "SGP":
		return "SGD"
	case "ID", "IDN":
		return "IDR"
	case "PH", "PHL":
		return "PHP"
	case "VN", "VNM":
		return "VND"
	case "GB", "UK":
		return "GBP"
	default:
		return "USD"
	}
}
