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

	updateAfter := normalizeLazadaDateFilter(updatedFrom, c.country)
	if updateAfter == "" {
		updateAfter = normalizeLazadaDateFilter(
			time.Now().UTC().Add(-defaultOrderLookback).Format(time.RFC3339),
			c.country,
		)
	}

	params := map[string]string{
		"app_key":        c.appKey,
		"access_token":   accessToken,
		"sign_method":    "sha256",
		"timestamp":      strconv.FormatInt(time.Now().UnixMilli(), 10),
		"offset":         strconv.Itoa(offset),
		"limit":          strconv.Itoa(limit),
		"update_after":   updateAfter,
		"sort_by":        "updated_at",
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
	OrderID            json.Number        `json:"order_id"`
	Price              json.Number        `json:"price"`
	Statuses           []string           `json:"statuses"`
	CreatedAt          string             `json:"created_at"`
	CustomerFirstName  string             `json:"customer_first_name"`
	CustomerLastName   string             `json:"customer_last_name"`
	AddressShipping    lazadaOrderAddress `json:"address_shipping"`
	AddressBilling     lazadaOrderAddress `json:"address_billing"`
}

type lazadaOrderAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
	City      string `json:"city"`
	PostCode  string `json:"post_code"`
	Country   string `json:"country"`
}

func (r *ordersGetResponse) toOrderList(country string) *response.OrderListResponse {
	total, _ := r.Data.CountTotal.Int64()
	if total == 0 {
		total, _ = r.Data.Count.Int64()
	}

	items := make([]response.OrderResponse, 0, len(r.Data.Orders))
	currency := currencyForCountry(country)

	for _, order := range r.Data.Orders {
		parsedAt := parseLazadaOrderTime(order.CreatedAt, country)
		shippingAddress := order.AddressShipping
		if shippingAddress.isEmpty() {
			shippingAddress = order.AddressBilling
		}

		item := response.OrderResponse{
			ExternalOrderID: order.OrderID.String(),
			Status:          formatOrderStatus(order.Statuses),
			TotalAmount:     order.Price.String(),
			Currency:        currency,
			CustomerName:    formatCustomerName(order.CustomerFirstName, order.CustomerLastName, shippingAddress),
			CustomerEmail:   strings.TrimSpace(shippingAddress.Email),
			CustomerPhone:   firstNonEmpty(strings.TrimSpace(shippingAddress.Phone), strings.TrimSpace(order.AddressBilling.Phone)),
			CustomerAddress: formatShippingAddress(shippingAddress),
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

func (a lazadaOrderAddress) isEmpty() bool {
	return strings.TrimSpace(a.FirstName) == "" &&
		strings.TrimSpace(a.LastName) == "" &&
		strings.TrimSpace(a.Address1) == "" &&
		strings.TrimSpace(a.City) == ""
}

func formatCustomerName(firstName, lastName string, address lazadaOrderAddress) string {
	name := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(firstName),
		strings.TrimSpace(lastName),
	}, " "))
	if name != "" {
		return name
	}
	return strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(address.FirstName),
		strings.TrimSpace(address.LastName),
	}, " "))
}

func formatShippingAddress(address lazadaOrderAddress) string {
	parts := make([]string, 0, 5)
	for _, part := range []string{
		strings.TrimSpace(address.Address1),
		strings.TrimSpace(address.Address2),
		strings.TrimSpace(address.City),
		strings.TrimSpace(address.PostCode),
		strings.TrimSpace(address.Country),
	} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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

func timezoneForCountry(country string) *time.Location {
	switch strings.ToLower(strings.TrimSpace(country)) {
	case "th", "id", "vn":
		return time.FixedZone("UTC+7", 7*3600)
	case "sg", "my", "ph":
		return time.FixedZone("UTC+8", 8*3600)
	default:
		return time.UTC
	}
}

func normalizeLazadaDateFilter(raw, country string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	loc := timezoneForCountry(country)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.In(loc).Format("2006-01-02T15:04:05-07:00")
		}
	}
	if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", raw, loc); err == nil {
		return parsed.Format("2006-01-02T15:04:05-07:00")
	}
	return raw
}

func parseLazadaOrderTime(raw, country string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}

	loc := timezoneForCountry(country)
	layouts := []string{
		time.RFC3339Nano,
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
	if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", raw, loc); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}
