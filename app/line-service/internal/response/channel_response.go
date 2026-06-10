package response

import (
	"time"

	"github.com/google/uuid"
)

type ChannelResponse struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	ExternalID     string     `json:"external_id"`
	DisplayName    string     `json:"display_name"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
