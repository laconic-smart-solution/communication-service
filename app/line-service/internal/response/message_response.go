package response

import (
	"time"

	"github.com/google/uuid"
)

type MessageResponse struct {
	ID          uuid.UUID `json:"id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Direction   string    `json:"direction"`
	RecipientID string    `json:"recipient_id"`
	Body        string    `json:"body"`
	Status      string    `json:"status"`
	ExternalID  string    `json:"external_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
