package request

type ConnectStoreRequest struct {
	Code        string `json:"code" validate:"required"`
	DisplayName string `json:"display_name"`
}

type RefreshStoreRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type AuthorizeURLRequest struct {
	State string `query:"state"`
}

type ListOrdersRequest struct {
	AccessToken string `query:"access_token" validate:"required"`
	SellerID    string `query:"seller_id"`
	UpdatedFrom string `query:"updated_from"`
	Page        int    `query:"page" validate:"omitempty,min=1"`
	Limit       int    `query:"limit" validate:"omitempty,min=1,max=100"`
}
