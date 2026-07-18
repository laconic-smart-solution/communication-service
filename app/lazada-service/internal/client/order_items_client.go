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

	var raw orderItemsGetResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if raw.Code != "" && raw.Code != "0" {
		return nil, fmt.Errorf("lazada api error: %s", raw.Message)
	}

	return groupLazadaOrderItems(raw.Data), nil
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

	var raw ordersItemsGetResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if raw.Code != "" && raw.Code != "0" {
		return nil, fmt.Errorf("lazada api error: %s", raw.Message)
	}

	result := make(map[string][]response.OrderItemResponse, len(raw.Data))
	for _, entry := range raw.Data {
		orderID := entry.OrderID.String()
		if orderID == "" {
			continue
		}
		result[orderID] = groupLazadaOrderItems(entry.OrderItems)
	}
	return result, nil
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

type ordersItemsGetResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		OrderID    json.Number       `json:"order_id"`
		OrderItems []lazadaOrderItem `json:"order_items"`
	} `json:"data"`
}

type orderItemsGetResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Data    []lazadaOrderItem `json:"data"`
}

type lazadaOrderItem struct {
	OrderID  json.Number `json:"order_id"`
	SKU      string      `json:"sku"`
	ShopSKU  string      `json:"shop_sku"`
	Name     string      `json:"name"`
	ItemPrice string     `json:"item_price"`
	PaidPrice string     `json:"paid_price"`
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
		priceStr := firstNonEmpty(strings.TrimSpace(item.ItemPrice), strings.TrimSpace(item.PaidPrice))
		if priceStr == "" {
			priceStr = "0"
		}
		key := lineItemGroupKey{sku: sku, productName: name, price: priceStr}
		acc, ok := groups[key]
		if !ok {
			priceRat := new(big.Rat)
			if _, ok := priceRat.SetString(priceStr); !ok {
				priceRat.SetInt64(0)
			}
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
