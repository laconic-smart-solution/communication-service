package response

import "time"

// TokenConnectResponse is returned by connect/refresh — stateless; caller persists tokens.
type TokenConnectResponse struct {
	SellerID         string     `json:"seller_id"`
	DisplayName      string     `json:"display_name"`
	Country          string     `json:"country"`
	Account          string     `json:"account"`
	ShortCode        string     `json:"short_code"`
	AccessToken      string     `json:"access_token"`
	RefreshToken     string     `json:"refresh_token"`
	TokenExpiresAt   *time.Time `json:"token_expires_at,omitempty"`
	RefreshExpiresAt *time.Time `json:"refresh_expires_at,omitempty"`
}

type AuthorizeURLResponse struct {
	URL string `json:"url"`
}
