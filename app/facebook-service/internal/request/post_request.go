package request

import "github.com/google/uuid"

type CreatePostRequest struct {
	ChannelID uuid.UUID `json:"channel_id" validate:"required"`
	Message   string    `json:"message" validate:"required"`
}

type UpdatePostRequest struct {
	Message *string `json:"message" validate:"required"`
}

type ListPostsRequest struct {
	ChannelID string `query:"channel_id"`
	Page      int    `query:"page" validate:"omitempty,min=1"`
	Limit     int    `query:"limit" validate:"omitempty,min=1,max=100"`
}

type ConnectChannelRequest struct {
	ExternalID  string `json:"external_id" validate:"required"`
	DisplayName string `json:"display_name"`
	AccessToken string `json:"access_token" validate:"required"`
}

type ListChannelsRequest struct {
	Page  int `query:"page" validate:"omitempty,min=1"`
	Limit int `query:"limit" validate:"omitempty,min=1,max=100"`
}
