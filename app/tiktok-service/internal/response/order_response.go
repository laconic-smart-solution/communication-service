package response

import "time"

type OrderResponse struct {
	ExternalOrderID    string              `json:"external_order_id"`
	Status             string              `json:"status"`
	TotalAmount        string              `json:"total_amount"`
	Currency           string              `json:"currency"`
	OrderedAt          *time.Time          `json:"ordered_at,omitempty"`
	CustomerName       string              `json:"customer_name,omitempty"`
	CustomerEmail      string              `json:"customer_email,omitempty"`
	CustomerPhone      string              `json:"customer_phone,omitempty"`
	CustomerAddress    string              `json:"customer_address,omitempty"`
	ExternalCustomerID string              `json:"external_customer_id,omitempty"`
	Items              []OrderItemResponse `json:"items"`
}

type OrderListResponse struct {
	Items []OrderResponse `json:"items"`
	Total int             `json:"total"`
}
