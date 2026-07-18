package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
)

const (
	ordersItemsGetPath = "/orders/items/get"
	orderItemsGetPath  = "/order/items/get"
	maxOrderIDsPerCall = 50
)

func (c *lazadaClientImpl) GetMultipleOrderItems(
	ctx context.Context,
	creds Credentials,
	accessToken string,
	orderIDs []string,
) (map[string][]response.OrderItemResponse, error) {
	result := make(map[string][]response.OrderItemResponse)
	ids := uniqueNonEmpty(orderIDs)
	if len(ids) == 0 {
		return result, nil
	}

	for start := 0; start < len(ids); start += maxOrderIDsPerCall {
		end := start + maxOrderIDsPerCall
		if end > len(ids) {
			end = len(ids)
		}
		batch, err := c.fetchMultipleOrderItems(ctx, creds, accessToken, ids[start:end])
		if err != nil {
			return nil, err
		}
		for orderID, items := range batch {
			result[orderID] = items
		}
	}
	return result, nil
}

func (c *lazadaClientImpl) GetOrderItems(
	ctx context.Context,
	creds Credentials,
	accessToken string,
	orderID string,
) ([]response.OrderItemResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, fmt.Errorf("order id is required")
	}

	params := map[string]string{
		"app_key":      creds.AppKey,
		"access_token": accessToken,
		"sign_method":  "sha256",
		"timestamp":    strconv.FormatInt(time.Now().UnixMilli(), 10),
		"order_id":     orderID,
	}
	params["sign"] = sign(orderItemsGetPath, params, creds.AppSecret)

	body, err := c.doLazadaGET(ctx, orderItemsGetPath, params)
	if err != nil {
		return nil, err
	}

	rawItems, err := parseOrderItemsGetBody(body)
	if err != nil {
		return nil, err
	}
	return groupLazadaOrderItems(rawItems), nil
}

func (c *lazadaClientImpl) fetchMultipleOrderItems(
	ctx context.Context,
	creds Credentials,
	accessToken string,
	orderIDs []string,
) (map[string][]response.OrderItemResponse, error) {
	params := map[string]string{
		"app_key":      creds.AppKey,
		"access_token": accessToken,
		"sign_method":  "sha256",
		"timestamp":    strconv.FormatInt(time.Now().UnixMilli(), 10),
		"order_ids":    formatOrderIDsParam(orderIDs),
	}
	params["sign"] = sign(ordersItemsGetPath, params, creds.AppSecret)

	body, err := c.doLazadaGET(ctx, ordersItemsGetPath, params)
	if err != nil {
		return nil, err
	}

	return parseOrdersItemsGetBody(body)
}

func (c *lazadaClientImpl) doLazadaGET(ctx context.Context, apiPath string, params map[string]string) ([]byte, error) {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}

	apiBase := regionalRESTBaseURL(c.country)
	reqURL := apiBase + apiPath + "?" + q.Encode()
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
	return body, nil
}

// flexibleString accepts JSON strings or numbers (Lazada mixes both).
type flexibleString string

func (f *flexibleString) UnmarshalJSON(b []byte) error {
	b = bytesTrimSpace(b)
	if string(b) == "null" || len(b) == 0 {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexibleString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexibleString(n.String())
	return nil
}

func (f flexibleString) String() string {
	return string(f)
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

type lazadaAPIEnvelope struct {
	Code    flexibleString  `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (e lazadaAPIEnvelope) ok() bool {
	code := strings.TrimSpace(e.Code.String())
	return code == "" || code == "0"
}

type lazadaOrderItem struct {
	OrderID   flexibleString `json:"order_id"`
	SKU       string         `json:"sku"`
	ShopSKU   string         `json:"shop_sku"`
	Name      string         `json:"name"`
	ItemPrice flexibleString `json:"item_price"`
	PaidPrice flexibleString `json:"paid_price"`
}

type multiOrderItemsEntry struct {
	OrderID    flexibleString    `json:"order_id"`
	OrderItems []lazadaOrderItem `json:"order_items"`
}

func parseOrdersItemsGetBody(body []byte) (map[string][]response.OrderItemResponse, error) {
	var env lazadaAPIEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode response: %w; body=%s", err, truncateBody(body))
	}
	if !env.ok() {
		return nil, fmt.Errorf("lazada api error: %s", firstNonEmpty(env.Message, env.Code.String()))
	}

	result := make(map[string][]response.OrderItemResponse)
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return result, nil
	}

	var entries []multiOrderItemsEntry
	if err := json.Unmarshal(env.Data, &entries); err != nil {
		return nil, fmt.Errorf("decode orders items data: %w; body=%s", err, truncateBody(body))
	}
	for _, entry := range entries {
		orderID := strings.TrimSpace(entry.OrderID.String())
		if orderID == "" {
			continue
		}
		result[orderID] = groupLazadaOrderItems(entry.OrderItems)
	}
	return result, nil
}

func parseOrderItemsGetBody(body []byte) ([]lazadaOrderItem, error) {
	var env lazadaAPIEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode response: %w; body=%s", err, truncateBody(body))
	}
	if !env.ok() {
		return nil, fmt.Errorf("lazada api error: %s", firstNonEmpty(env.Message, env.Code.String()))
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return []lazadaOrderItem{}, nil
	}

	// Common shape: data is []orderItem
	var items []lazadaOrderItem
	if err := json.Unmarshal(env.Data, &items); err == nil {
		return items, nil
	}

	// Alternate: data is { order_items: [...] }
	var wrapped struct {
		OrderItems []lazadaOrderItem `json:"order_items"`
	}
	if err := json.Unmarshal(env.Data, &wrapped); err == nil && len(wrapped.OrderItems) > 0 {
		return wrapped.OrderItems, nil
	}

	return nil, fmt.Errorf("decode order items data: unexpected shape; body=%s", truncateBody(body))
}

func truncateBody(body []byte) string {
	const max = 500
	s := strings.TrimSpace(string(body))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

type lineItemGroupKey struct {
	sku         string
	productName string
	price       string
}

func groupLazadaOrderItems(rawItems []lazadaOrderItem) []response.OrderItemResponse {
	if len(rawItems) == 0 {
		return []response.OrderItemResponse{}
	}

	type accumulator struct {
		sku         string
		productName string
		price       *big.Rat
		quantity    int
	}

	groups := make(map[lineItemGroupKey]*accumulator)
	orderKeys := make([]lineItemGroupKey, 0)

	for _, item := range rawItems {
		sku := firstNonEmpty(strings.TrimSpace(item.SKU), strings.TrimSpace(item.ShopSKU))
		name := strings.TrimSpace(item.Name)
		priceStr := firstNonEmpty(
			strings.TrimSpace(item.ItemPrice.String()),
			strings.TrimSpace(item.PaidPrice.String()),
		)
		priceRat := new(big.Rat)
		if priceStr == "" {
			priceRat.SetInt64(0)
		} else if _, ok := priceRat.SetString(priceStr); !ok {
			priceRat.SetInt64(0)
		}
		normalizedPrice := formatRatAmount(priceRat)
		key := lineItemGroupKey{sku: sku, productName: name, price: normalizedPrice}
		acc, ok := groups[key]
		if !ok {
			acc = &accumulator{
				sku:         sku,
				productName: name,
				price:       priceRat,
				quantity:    0,
			}
			groups[key] = acc
			orderKeys = append(orderKeys, key)
		}
		acc.quantity++
	}

	items := make([]response.OrderItemResponse, 0, len(orderKeys))
	for _, key := range orderKeys {
		acc := groups[key]
		qty := new(big.Rat).SetInt64(int64(acc.quantity))
		lineTotal := new(big.Rat).Mul(acc.price, qty)
		items = append(items, response.OrderItemResponse{
			SKU:         acc.sku,
			ProductName: acc.productName,
			Price:       formatRatAmount(acc.price),
			Quantity:    acc.quantity,
			LineTotal:   formatRatAmount(lineTotal),
		})
	}
	return items
}

func formatRatAmount(value *big.Rat) string {
	if value == nil {
		return "0"
	}
	return value.FloatString(2)
}

func formatOrderIDsParam(orderIDs []string) string {
	parts := make([]string, 0, len(orderIDs))
	for _, id := range orderIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			parts = append(parts, id)
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}
