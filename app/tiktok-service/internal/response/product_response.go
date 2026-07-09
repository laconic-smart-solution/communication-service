package response

type ProductResponse struct {
	ExternalProductID string `json:"external_product_id"`
	SKU               string `json:"sku"`
	Name              string `json:"name"`
	Price             string `json:"price"`
	Currency          string `json:"currency"`
	Status            string `json:"status"`
}

type ProductListResponse struct {
	Items []ProductResponse `json:"items"`
	Total int             `json:"total"`
}
