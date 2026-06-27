package mapper

import (
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
)

func ToStoreResponse(store models.Store) response.StoreResponse {
	return response.StoreResponse{
		ID:               store.ID,
		TenantID:         store.TenantID,
		SellerID:         store.SellerID,
		DisplayName:      store.DisplayName,
		Country:          store.Country,
		Account:          store.Account,
		ShortCode:        store.ShortCode,
		TokenExpiresAt:   store.TokenExpiresAt,
		RefreshExpiresAt: store.RefreshExpiresAt,
		CreatedAt:        store.CreatedAt,
	}
}

func ToStoreResponses(items []models.Store) []response.StoreResponse {
	out := make([]response.StoreResponse, 0, len(items))
	for _, store := range items {
		out = append(out, ToStoreResponse(store))
	}
	return out
}
