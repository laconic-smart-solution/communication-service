package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
)

const (
	ordersGetPath        = "/orders/get"
	defaultOrderLookback = 90 * 24 * time.Hour
)

func (c *lazadaClientImpl) ListOrders(
	ctx context.Context,
	accessToken string,
	page, limit int,
	updatedFrom string,
) (*response.OrderListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	updateAfter := strings.TrimSpace(updatedFrom)
	if updateAfter == "" {
		updateAfter = time.Now().UTC().Add(-defaultOrderLookback).Format(time.RFC3339)
	}

	params := map[string]string{
		"app_key":      c.appKey,
		"access_token": accessToken,
		"sign_method":  "sha256",
		"timestamp":    strconv.FormatInt(time.Now().UnixMilli(), 10),
		"offset":       strconv.Itoa(offset),
		"limit":        strconv.Itoa(limit),
		"update_after": updateAfter,
		"sort_by":      "updated_at",
		"sort_direction": "DESC",
	}
	params["sign"] = sign(ordersGetPath, params, c.appSecret)

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}

	apiBase := regionalRESTBaseURL(c.country)
	reqURL := apiBase + ordersGetPath + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

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

	var raw ordersGetResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if raw.Code != "" && raw.Code != "0" {
		return nil, fmt.Errorf("lazada api error: %s", raw.Message)
	}

	return raw.toOrderList(c.country), nil
}

type ordersGetResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		CountTotal json.Number   `json:"countTotal"`
		Count      json.Number   `json:"count"`
		Orders     []lazadaOrder `json:"orders"`
	} `json:"data"`
}

type lazadaOrder struct {
	OrderID   json.Number `json:"order_id"`
	Price     json.Number `json:"price"`
	Statuses  []string    `json:"statuses"`
	CreatedAt string      `json:"created_at"`
}

func (r *ordersGetResponse) toOrderList(country string) *response.OrderListResponse {
	total, _ := r.Data.CountTotal.Int64()
	if total == 0 {
		total, _ = r.Data.Count.Int64()
	}

	items := make([]response.OrderResponse, 0, len(r.Data.Orders))
	currency := currencyForCountry(country)

	for _, order := range r.Data.Orders {
		parsedAt := parseLazadaOrderTime(order.CreatedAt)
		item := response.OrderResponse{
			ExternalOrderID: order.OrderID.String(),
			Status:          formatOrderStatus(order.Statuses),
			TotalAmount:     order.Price.String(),
			Currency:        currency,
		}
		if !parsedAt.IsZero() {
			item.OrderedAt = &parsedAt
		}
		items = append(items, item)
	}

	return &response.OrderListResponse{
		Items: items,
		Total: int(total),
	}
}

func formatOrderStatus(statuses []string) string {
	if len(statuses) == 0 {
		return ""
	}
	parts := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if trimmed := strings.TrimSpace(status); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, ",")
}

func parseLazadaOrderTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
