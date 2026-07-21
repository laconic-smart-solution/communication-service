package response

type OrderItemResponse struct {
	SKU         string `json:"sku"`
	ProductName string `json:"product_name"`
	Price       string `json:"price"`
	Quantity    int    `json:"quantity"`
	LineTotal   string `json:"line_total"`
	Currency    string `json:"currency,omitempty"`
	BuyerID     string `json:"buyer_id,omitempty"`
}
