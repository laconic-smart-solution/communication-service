package response

import (
	"time"

	"github.com/google/uuid"
)

type PostResponse struct {
	ID         uuid.UUID `json:"id"`
	ChannelID  uuid.UUID `json:"channel_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Message    string    `json:"message"`
	Status     string    `json:"status"`
	ExternalID string    `json:"external_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ChannelResponse struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	ExternalID  string    `json:"external_id"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}
