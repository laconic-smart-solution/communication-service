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

const productsGetPath = "/products/get"

func (c *lazadaClientImpl) ListProducts(
	ctx context.Context,
	creds Credentials,
	accessToken string,
	page, limit int,
	search string,
) (*response.ProductListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	offset := (page - 1) * limit

	params := map[string]string{
		"app_key":      creds.AppKey,
		"access_token": accessToken,
		"sign_method":  "sha256",
		"timestamp":    strconv.FormatInt(time.Now().UnixMilli(), 10),
		"filter":       "all",
		"offset":       strconv.Itoa(offset),
		"limit":        strconv.Itoa(limit),
		"options":      "1",
	}
	if search != "" {
		params["search"] = search
	}
	params["sign"] = sign(productsGetPath, params, creds.AppSecret)

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}

	apiBase := regionalRESTBaseURL(c.country)
	reqURL := apiBase + productsGetPath + "?" + q.Encode()
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

	var raw productsGetResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if raw.Code != "" && raw.Code != "0" {
		return nil, fmt.Errorf("lazada api error: %s", raw.Message)
	}

	return raw.toProductList(c.country), nil
}

func regionalRESTBaseURL(country string) string {
	switch strings.ToLower(strings.TrimSpace(country)) {
	case "th":
		return "https://api.lazada.co.th/rest"
	case "sg":
		return "https://api.lazada.sg/rest"
	case "my":
		return "https://api.lazada.com.my/rest"
	case "ph":
		return "https://api.lazada.com.ph/rest"
	case "id":
		return "https://api.lazada.co.id/rest"
	case "vn":
		return "https://api.lazada.vn/rest"
	default:
		return "https://api.lazada.com/rest"
	}
}

type productsGetResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TotalProducts json.Number       `json:"total_products"`
		Products      []lazadaProduct   `json:"products"`
	} `json:"data"`
}

type lazadaProduct struct {
	ItemID     json.Number     `json:"item_id"`
	Status     string          `json:"status"`
	Attributes json.RawMessage `json:"attributes"`
	Skus       []lazadaSku     `json:"skus"`
}

type lazadaSku struct {
	SellerSku string      `json:"SellerSku"`
	ShopSku   string      `json:"ShopSku"`
	Price     json.Number `json:"price"`
	Status    string      `json:"Status"`
}

func (r *productsGetResponse) toProductList(country string) *response.ProductListResponse {
	total, _ := r.Data.TotalProducts.Int64()
	items := make([]response.ProductResponse, 0, len(r.Data.Products))
	currency := currencyForCountry(country)

	for _, product := range r.Data.Products {
		name := parseProductName(product.Attributes)
		status := strings.TrimSpace(product.Status)
		externalID := product.ItemID.String()

		if len(product.Skus) == 0 {
			items = append(items, response.ProductResponse{
				ExternalProductID: externalID,
				Name:              name,
				Currency:          currency,
				Status:            status,
			})
			continue
		}

		for _, sku := range product.Skus {
			skuCode := sku.SellerSku
			if skuCode == "" {
				skuCode = sku.ShopSku
			}
			skuStatus := strings.TrimSpace(sku.Status)
			if skuStatus == "" {
				skuStatus = status
			}
			items = append(items, response.ProductResponse{
				ExternalProductID: externalID,
				SKU:               skuCode,
				Name:              name,
				Price:             sku.Price.String(),
				Currency:          currency,
				Status:            skuStatus,
			})
		}
	}

	return &response.ProductListResponse{
		Items: items,
		Total: int(total),
	}
}

func parseProductName(attributes json.RawMessage) string {
	if len(attributes) == 0 {
		return ""
	}
	var attrs map[string]string
	if err := json.Unmarshal(attributes, &attrs); err == nil {
		if name := strings.TrimSpace(attrs["name"]); name != "" {
			return name
		}
	}
	var node map[string]json.RawMessage
	if err := json.Unmarshal(attributes, &node); err == nil {
		if raw, ok := node["name"]; ok {
			var name string
			if err := json.Unmarshal(raw, &name); err == nil {
				return strings.TrimSpace(name)
			}
		}
	}
	return ""
}

func currencyForCountry(country string) string {
	switch strings.ToLower(strings.TrimSpace(country)) {
	case "th":
		return "THB"
	case "sg":
		return "SGD"
	case "my":
		return "MYR"
	case "ph":
		return "PHP"
	case "id":
		return "IDR"
	case "vn":
		return "VND"
	default:
		return ""
	}
}
