package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/response"
)

func (c *tiktokClientImpl) ListProducts(
	ctx context.Context,
	creds Credentials,
	accessToken, shopID string,
	page, limit int,
	search string,
) (*response.ProductListResponse, error) {
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

	currency := currencyForRegion(shop.Region)
	pageToken := ""
	var lastPage *response.ProductListResponse
	var total int

	for current := 1; current <= page; current++ {
		query := map[string]string{
			"page_size": strconv.Itoa(limit),
		}
		if pageToken != "" {
			query["page_token"] = pageToken
		}

		body := map[string]any{}
		if trimmed := strings.TrimSpace(search); trimmed != "" {
			body["seller_skus"] = []string{trimmed}
		}

		path := "/product/" + apiVersion + "/products/search"
		data, err := c.doSignedJSON(ctx, creds, http.MethodPost, path, accessToken, shop.Cipher, query, body)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Products []tiktokProductSummary `json:"products"`
			Total    int                    `json:"total_count"`
			Next     string                 `json:"next_page_token"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf("decode products: %w", err)
		}
		total = payload.Total

		items := make([]response.ProductResponse, 0, len(payload.Products))
		for _, p := range payload.Products {
			items = append(items, p.toProductResponse(currency))
		}
		lastPage = &response.ProductListResponse{Items: items, Total: total}
		pageToken = strings.TrimSpace(payload.Next)
		if pageToken == "" && current < page {
			return &response.ProductListResponse{Items: []response.ProductResponse{}, Total: total}, nil
		}
	}

	if lastPage == nil {
		return &response.ProductListResponse{Items: []response.ProductResponse{}, Total: 0}, nil
	}
	return lastPage, nil
}

func (c *tiktokClientImpl) GetProduct(
	ctx context.Context,
	creds Credentials,
	accessToken, shopID, externalProductID string,
) (*response.ProductDetailResponse, error) {
	productID := strings.TrimSpace(externalProductID)
	if productID == "" {
		return nil, ErrNotFound
	}

	shop, err := c.ResolveShopCipher(ctx, creds, accessToken, shopID)
	if err != nil {
		return nil, err
	}

	path := "/product/" + apiVersion + "/products/" + productID
	data, err := c.doSignedJSON(ctx, creds, http.MethodGet, path, accessToken, shop.Cipher, nil, nil)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var detail tiktokProductDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("decode product detail: %w", err)
	}
	if strings.TrimSpace(detail.ID) == "" {
		return nil, ErrNotFound
	}
	return detail.toDetailResponse(currencyForRegion(shop.Region)), nil
}

type tiktokProductSummary struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Skus   []struct {
		ID         string `json:"id"`
		SellerSku  string `json:"seller_sku"`
		Price      tiktokMoney `json:"price"`
	} `json:"skus"`
}

type tiktokMoney struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func (p tiktokProductSummary) toProductResponse(fallbackCurrency string) response.ProductResponse {
	sku := ""
	price := ""
	currency := fallbackCurrency
	if len(p.Skus) > 0 {
		sku = firstNonEmpty(p.Skus[0].SellerSku, p.Skus[0].ID)
		price = strings.TrimSpace(p.Skus[0].Price.Amount)
		if c := strings.TrimSpace(p.Skus[0].Price.Currency); c != "" {
			currency = strings.ToUpper(c)
		}
	}
	return response.ProductResponse{
		ExternalProductID: strings.TrimSpace(p.ID),
		SKU:               sku,
		Name:              strings.TrimSpace(p.Title),
		Price:             price,
		Currency:          currency,
		Status:            mapProductStatus(p.Status),
	}
}

type tiktokProductDetail struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	MainImages  []struct {
		URLs []string `json:"urls"`
	} `json:"main_images"`
	Skus []struct {
		ID        string      `json:"id"`
		SellerSku string      `json:"seller_sku"`
		Price     tiktokMoney `json:"price"`
		Inventory []struct {
			Quantity int `json:"quantity"`
		} `json:"inventory"`
	} `json:"skus"`
}

func (p tiktokProductDetail) toDetailResponse(fallbackCurrency string) *response.ProductDetailResponse {
	images := make([]string, 0)
	for _, img := range p.MainImages {
		for _, u := range img.URLs {
			if trimmed := strings.TrimSpace(u); trimmed != "" {
				images = append(images, trimmed)
			}
		}
	}

	variants := make([]response.ProductVariantResponse, 0, len(p.Skus))
	sku := ""
	price := ""
	currency := fallbackCurrency
	for i, s := range p.Skus {
		variantSKU := firstNonEmpty(s.SellerSku, s.ID)
		variantPrice := strings.TrimSpace(s.Price.Amount)
		variantCurrency := fallbackCurrency
		if c := strings.TrimSpace(s.Price.Currency); c != "" {
			variantCurrency = strings.ToUpper(c)
		}
		qty := 0
		for _, inv := range s.Inventory {
			qty += inv.Quantity
		}
		variants = append(variants, response.ProductVariantResponse{
			SKU:      variantSKU,
			Price:    variantPrice,
			Currency: variantCurrency,
			Status:   mapProductStatus(p.Status),
			Quantity: strconv.Itoa(qty),
		})
		if i == 0 {
			sku = variantSKU
			price = variantPrice
			currency = variantCurrency
		}
	}

	return &response.ProductDetailResponse{
		ExternalProductID: strings.TrimSpace(p.ID),
		SKU:               sku,
		Name:              strings.TrimSpace(p.Title),
		Description:       strings.TrimSpace(p.Description),
		Price:             price,
		Currency:          currency,
		Status:            mapProductStatus(p.Status),
		Images:            images,
		Variants:          variants,
	}
}

func mapProductStatus(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "ACTIVATE", "LIVE", "ACTIVE":
		return "active"
	case "DRAFT":
		return "draft"
	case "SELLER_DEACTIVATED", "PLATFORM_DEACTIVATED", "FREEZE", "DELETED":
		return "inactive"
	default:
		if raw == "" {
			return "unknown"
		}
		return strings.ToLower(raw)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
