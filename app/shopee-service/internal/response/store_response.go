package response

import (
	"time"

	"github.com/google/uuid"
)

type StoreResponse struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	SellerID         string     `json:"seller_id"`
	DisplayName      string     `json:"display_name"`
	Country          string     `json:"country"`
	Account          string     `json:"account"`
	ShortCode        string     `json:"short_code"`
	TokenExpiresAt   *time.Time `json:"token_expires_at,omitempty"`
	RefreshExpiresAt *time.Time `json:"refresh_expires_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type AuthorizeURLResponse struct {
	URL string `json:"url"`
}
