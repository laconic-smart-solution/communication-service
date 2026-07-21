package request

type SendBatchMessageRequest struct {
	RecipientIDs []string `json:"recipient_ids" validate:"required,min=1,dive,required"`
	Text         string   `json:"text" validate:"required"`
}
