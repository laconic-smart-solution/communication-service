package response

type ProductVariantResponse struct {
	SKU      string `json:"sku"`
	Price    string `json:"price"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
	Quantity string `json:"quantity,omitempty"`
}

type ProductDetailResponse struct {
	ExternalProductID string                   `json:"external_product_id"`
	SKU               string                   `json:"sku"`
	Name              string                   `json:"name"`
	Description       string                   `json:"description"`
	Price             string                   `json:"price"`
	Currency          string                   `json:"currency"`
	Status            string                   `json:"status"`
	Images            []string                 `json:"images"`
	Variants          []ProductVariantResponse `json:"variants"`
}
