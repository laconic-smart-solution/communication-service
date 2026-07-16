package response

type QRResponse struct {
	BasicID      string `json:"basic_id"`
	AddFriendURL string `json:"add_friend_url"`
	QRImage      string `json:"qr_image"`
}
