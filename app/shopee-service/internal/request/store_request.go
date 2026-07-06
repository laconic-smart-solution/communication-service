package request

type ConnectStoreRequest struct {
	Code        string `json:"code" validate:"required"`
	ShopID      string `json:"shop_id"`
	DisplayName string `json:"display_name"`
}

type ListStoresRequest struct {
	Page  int `query:"page" validate:"omitempty,min=1"`
	Limit int `query:"limit" validate:"omitempty,min=1,max=100"`
}

type AuthorizeURLRequest struct {
	State string `query:"state"`
}

type ListOrdersRequest struct {
	StoreID     string `query:"store_id" validate:"required,uuid"`
	UpdatedFrom string `query:"updated_from"`
	Page        int    `query:"page" validate:"omitempty,min=1"`
	Limit       int    `query:"limit" validate:"omitempty,min=1,max=100"`
}
