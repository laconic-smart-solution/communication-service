package request

type SendMessageRequest struct {
	RecipientID string `json:"recipient_id" validate:"required"`
	Text        string `json:"text" validate:"required"`
}
