package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/response"
)

func (c *tiktokClientImpl) ListOrders(
	ctx context.Context,
	creds Credentials,
	accessToken, shopID string,
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

	shop, err := c.ResolveShopCipher(ctx, creds, accessToken, shopID)
	if err != nil {
		return nil, err
	}

	updateGe := parseUpdatedFromUnix(updatedFrom)
	pageToken := ""
	var lastIDs []string
	var total int

	for current := 1; current <= page; current++ {
		query := map[string]string{
			"page_size": strconv.Itoa(limit),
		}
		if pageToken != "" {
			query["page_token"] = pageToken
		}

		body := map[string]any{
			"update_time_ge": updateGe,
		}

		path := "/order/" + apiVersion + "/orders/search"
		data, err := c.doSignedJSON(ctx, creds, http.MethodPost, path, accessToken, shop.Cipher, query, body)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Orders []struct {
				ID string `json:"id"`
			} `json:"orders"`
			Total int    `json:"total_count"`
			Next  string `json:"next_page_token"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf("decode order search: %w", err)
		}
		total = payload.Total
		lastIDs = make([]string, 0, len(payload.Orders))
		for _, o := range payload.Orders {
			if id := strings.TrimSpace(o.ID); id != "" {
				lastIDs = append(lastIDs, id)
			}
		}
		pageToken = strings.TrimSpace(payload.Next)
		if pageToken == "" && current < page {
			return &response.OrderListResponse{Items: []response.OrderResponse{}, Total: total}, nil
		}
	}

	if len(lastIDs) == 0 {
		return &response.OrderListResponse{Items: []response.OrderResponse{}, Total: total}, nil
	}

	details, err := c.getOrderDetails(ctx, creds, accessToken, shop.Cipher, lastIDs)
	if err != nil {
		return nil, err
	}

	fallbackCurrency := currencyForRegion(shop.Region)
	items := make([]response.OrderResponse, 0, len(details))
	for _, detail := range details {
		items = append(items, detail.toOrderResponse(fallbackCurrency))
	}
	return &response.OrderListResponse{Items: items, Total: total}, nil
}

func (c *tiktokClientImpl) getOrderDetails(
	ctx context.Context,
	creds Credentials,
	accessToken, shopCipher string,
	orderIDs []string,
) ([]tiktokOrderDetail, error) {
	const batchSize = 50
	out := make([]tiktokOrderDetail, 0, len(orderIDs))

	for start := 0; start < len(orderIDs); start += batchSize {
		end := start + batchSize
		if end > len(orderIDs) {
			end = len(orderIDs)
		}
		batch := orderIDs[start:end]
		query := map[string]string{
			"ids": strings.Join(batch, ","),
		}
		path := "/order/" + apiVersion + "/orders"
		data, err := c.doSignedJSON(ctx, creds, http.MethodGet, path, accessToken, shopCipher, query, nil)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Orders []tiktokOrderDetail `json:"orders"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf("decode order details: %w", err)
		}
		out = append(out, payload.Orders...)
	}
	return out, nil
}

type tiktokOrderDetail struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	CreateTime       int64  `json:"create_time"`
	BuyerEmail       string `json:"buyer_email"`
	UserID           string `json:"user_id"`
	Payment          tiktokOrderPayment `json:"payment"`
	RecipientAddress tiktokRecipientAddress `json:"recipient_address"`
	LineItems        []tiktokLineItem `json:"line_items"`
}

type tiktokOrderPayment struct {
	Currency    string `json:"currency"`
	TotalAmount string `json:"total_amount"`
}

type tiktokRecipientAddress struct {
	Name          string `json:"name"`
	PhoneNumber   string `json:"phone_number"`
	FullAddress   string `json:"full_address"`
	AddressDetail string `json:"address_detail"`
	AddressLine1  string `json:"address_line1"`
	AddressLine2  string `json:"address_line2"`
	PostalCode    string `json:"postal_code"`
	RegionCode    string `json:"region_code"`
}

type tiktokLineItem struct {
	ID          string `json:"id"`
	SKUID       string `json:"sku_id"`
	SellerSKU   string `json:"seller_sku"`
	ProductName string `json:"product_name"`
	SKUName     string `json:"sku_name"`
	SalePrice   string `json:"sale_price"`
	Currency    string `json:"currency"`
}

func (o tiktokOrderDetail) toOrderResponse(fallbackCurrency string) response.OrderResponse {
	currency := strings.ToUpper(strings.TrimSpace(o.Payment.Currency))
	if currency == "" {
		currency = fallbackCurrency
	}

	items := make([]response.OrderItemResponse, 0, len(o.LineItems))
	for _, line := range o.LineItems {
		itemCurrency := strings.ToUpper(strings.TrimSpace(line.Currency))
		if itemCurrency == "" {
			itemCurrency = currency
		}
		price := strings.TrimSpace(line.SalePrice)
		sku := firstNonEmpty(line.SellerSKU, line.SKUID, line.ID)
		name := firstNonEmpty(line.ProductName, line.SKUName, sku)
		items = append(items, response.OrderItemResponse{
			SKU:         sku,
			ProductName: name,
			Price:       price,
			Quantity:    1,
			LineTotal:   price,
			Currency:    itemCurrency,
			BuyerID:     strings.TrimSpace(o.UserID),
		})
	}

	var orderedAt *time.Time
	if o.CreateTime > 0 {
		t := time.Unix(o.CreateTime, 0).UTC()
		orderedAt = &t
	}

	addr := o.RecipientAddress
	return response.OrderResponse{
		ExternalOrderID:    strings.TrimSpace(o.ID),
		Status:             strings.TrimSpace(o.Status),
		TotalAmount:        strings.TrimSpace(o.Payment.TotalAmount),
		Currency:           currency,
		OrderedAt:          orderedAt,
		CustomerName:       strings.TrimSpace(addr.Name),
		CustomerEmail:      strings.TrimSpace(o.BuyerEmail),
		CustomerPhone:      strings.TrimSpace(addr.PhoneNumber),
		CustomerAddress:    formatTikTokAddress(addr),
		ExternalCustomerID: strings.TrimSpace(o.UserID),
		Items:              items,
	}
}

func formatTikTokAddress(addr tiktokRecipientAddress) string {
	if full := strings.TrimSpace(addr.FullAddress); full != "" {
		return full
	}
	parts := make([]string, 0, 5)
	for _, part := range []string{
		addr.AddressDetail,
		addr.AddressLine1,
		addr.AddressLine2,
		addr.PostalCode,
		addr.RegionCode,
	} {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return strings.Join(parts, ", ")
}

func parseUpdatedFromUnix(updatedFrom string) int64 {
	trimmed := strings.TrimSpace(updatedFrom)
	if trimmed == "" {
		return time.Now().Add(-90 * 24 * time.Hour).Unix()
	}
	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return t.Unix()
	}
	if t, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return t.Unix()
	}
	if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil && n > 0 {
		return n
	}
	return time.Now().Add(-90 * 24 * time.Hour).Unix()
}
