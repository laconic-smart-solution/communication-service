package mapper

import (
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/response"
)

func ToMessageResponse(m models.Message) response.MessageResponse {
	return response.MessageResponse{
		ID:          m.ID,
		ChannelID:   m.ChannelID,
		TenantID:    m.TenantID,
		Direction:   m.Direction,
		RecipientID: m.RecipientID,
		Body:        m.Body,
		Status:      m.Status,
		ExternalID:  m.ExternalID,
		CreatedAt:   m.CreatedAt,
	}
}
