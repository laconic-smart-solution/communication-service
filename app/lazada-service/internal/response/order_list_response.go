package response

type OrderListResponse struct {
	Items []OrderResponse `json:"items"`
	Total int             `json:"total"`
}
