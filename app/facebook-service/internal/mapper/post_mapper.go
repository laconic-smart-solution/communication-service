package mapper

import (
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/response"
)

func ToPostResponse(p models.Post) response.PostResponse {
	return response.PostResponse{
		ID:         p.ID,
		ChannelID:  p.ChannelID,
		TenantID:   p.TenantID,
		Message:    p.Message,
		Status:     p.Status,
		ExternalID: p.ExternalID,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

func ToPostResponses(items []models.Post) []response.PostResponse {
	out := make([]response.PostResponse, 0, len(items))
	for _, p := range items {
		out = append(out, ToPostResponse(p))
	}
	return out
}

func ToChannelResponse(ch models.Channel) response.ChannelResponse {
	return response.ChannelResponse{
		ID:          ch.ID,
		TenantID:    ch.TenantID,
		ExternalID:  ch.ExternalID,
		DisplayName: ch.DisplayName,
		CreatedAt:   ch.CreatedAt,
	}
}

func ToChannelResponses(items []models.Channel) []response.ChannelResponse {
	out := make([]response.ChannelResponse, 0, len(items))
	for _, ch := range items {
		out = append(out, ToChannelResponse(ch))
	}
	return out
}
