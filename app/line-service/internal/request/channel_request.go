package request

type ConnectChannelRequest struct {
	ExternalID  string `json:"external_id" validate:"required"`
	DisplayName string `json:"display_name"`
	AccessToken string `json:"access_token" validate:"required"`
}

type ListChannelsRequest struct {
	Page  int `query:"page" validate:"omitempty,min=1"`
	Limit int `query:"limit" validate:"omitempty,min=1,max=100"`
}
