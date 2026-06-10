package request

import "github.com/google/uuid"

type SendMessageRequest struct {
	ChannelID   uuid.UUID `json:"channel_id" validate:"required"`
	RecipientID string    `json:"recipient_id" validate:"required"`
	Text        string    `json:"text" validate:"required"`
}
