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

const productItemGetPath = "/product/item/get"

func (c *lazadaClientImpl) GetProduct(
	ctx context.Context,
	creds Credentials,
	accessToken string,
	externalProductID string,
	sku string,
) (*response.ProductDetailResponse, error) {
	itemID := strings.TrimSpace(externalProductID)
	if itemID == "" {
		return nil, fmt.Errorf("external product id is required")
	}

	params := map[string]string{
		"app_key":      creds.AppKey,
		"access_token": accessToken,
		"sign_method":  "sha256",
		"timestamp":    strconv.FormatInt(time.Now().UnixMilli(), 10),
		"item_id":      itemID,
	}
	params["sign"] = sign(productItemGetPath, params, creds.AppSecret)

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}

	apiBase := regionalRESTBaseURL(c.country)
	reqURL := apiBase + productItemGetPath + "?" + q.Encode()
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

	var raw productItemGetResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if raw.Code != "" && raw.Code != "0" {
		return nil, fmt.Errorf("lazada api error: %s", raw.Message)
	}

	return raw.toProductDetail(c.country, sku), nil
}

type productItemGetResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ItemID     json.Number       `json:"item_id"`
		Status     string            `json:"status"`
		Attributes json.RawMessage   `json:"attributes"`
		Images     []string          `json:"images"`
		Skus       []lazadaDetailSku `json:"skus"`
	} `json:"data"`
}

type lazadaDetailSku struct {
	SellerSku string          `json:"SellerSku"`
	ShopSku   string          `json:"ShopSku"`
	Price     json.Number     `json:"price"`
	Quantity  json.Number     `json:"quantity"`
	Status    string          `json:"Status"`
	Images    json.RawMessage `json:"Images"`
}

func (r *productItemGetResponse) toProductDetail(country, selectedSku string) *response.ProductDetailResponse {
	currency := currencyForCountry(country)
	name := parseProductName(r.Data.Attributes)
	description := parseProductDescription(r.Data.Attributes)
	status := strings.TrimSpace(r.Data.Status)
	externalID := r.Data.ItemID.String()

	variants := make([]response.ProductVariantResponse, 0, len(r.Data.Skus))
	for _, skuRow := range r.Data.Skus {
		skuCode := strings.TrimSpace(skuRow.SellerSku)
		if skuCode == "" {
			skuCode = strings.TrimSpace(skuRow.ShopSku)
		}
		skuStatus := strings.TrimSpace(skuRow.Status)
		if skuStatus == "" {
			skuStatus = status
		}
		variants = append(variants, response.ProductVariantResponse{
			SKU:      skuCode,
			Price:    skuRow.Price.String(),
			Currency: currency,
			Status:   skuStatus,
			Quantity: skuRow.Quantity.String(),
		})
	}

	selected := strings.TrimSpace(selectedSku)
	mainSku := selected
	mainPrice := ""
	mainStatus := status
	if mainSku != "" {
		for _, variant := range variants {
			if variant.SKU == mainSku {
				mainPrice = variant.Price
				mainStatus = variant.Status
				break
			}
		}
	} else if len(variants) > 0 {
		mainSku = variants[0].SKU
		mainPrice = variants[0].Price
		mainStatus = variants[0].Status
	}

	images := append([]string{}, r.Data.Images...)
	if len(images) == 0 {
		for _, skuRow := range r.Data.Skus {
			images = append(images, parseSkuImages(skuRow.Images)...)
		}
	}
	images = uniqueNonEmpty(images)

	return &response.ProductDetailResponse{
		ExternalProductID: externalID,
		SKU:               mainSku,
		Name:              name,
		Description:       description,
		Price:             mainPrice,
		Currency:          currency,
		Status:            mainStatus,
		Images:            images,
		Variants:          variants,
	}
}

func parseProductDescription(attributes json.RawMessage) string {
	if len(attributes) == 0 {
		return ""
	}
	var attrs map[string]string
	if err := json.Unmarshal(attributes, &attrs); err == nil {
		for _, key := range []string{"description", "short_description", "desc"} {
			if value := strings.TrimSpace(attrs[key]); value != "" {
				return value
			}
		}
	}
	return ""
}

func parseSkuImages(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var images []string
	if err := json.Unmarshal(raw, &images); err == nil {
		return images
	}
	var wrapped []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		result := make([]string, 0, len(wrapped))
		for _, image := range wrapped {
			if image.URL != "" {
				result = append(result, image.URL)
			}
		}
		return result
	}
	return nil
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
