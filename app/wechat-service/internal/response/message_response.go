package response

type SendMessageResponse struct {
	RecipientID string `json:"recipient_id"`
	Status      string `json:"status"`
}
