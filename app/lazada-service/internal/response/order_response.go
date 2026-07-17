package response

import "time"

type OrderResponse struct {
	ExternalOrderID string     `json:"external_order_id"`
	Status          string     `json:"status"`
	TotalAmount     string     `json:"total_amount"`
	Currency        string     `json:"currency"`
	OrderedAt       *time.Time `json:"ordered_at,omitempty"`
}
