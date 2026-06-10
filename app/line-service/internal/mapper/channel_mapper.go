package mapper

import (
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/response"
)

func ToChannelResponse(ch models.Channel) response.ChannelResponse {
	return response.ChannelResponse{
		ID:             ch.ID,
		TenantID:       ch.TenantID,
		ExternalID:     ch.ExternalID,
		DisplayName:    ch.DisplayName,
		TokenExpiresAt: ch.TokenExpiresAt,
		CreatedAt:      ch.CreatedAt,
	}
}

func ToChannelResponses(items []models.Channel) []response.ChannelResponse {
	out := make([]response.ChannelResponse, 0, len(items))
	for _, ch := range items {
		out = append(out, ToChannelResponse(ch))
	}
	return out
}
