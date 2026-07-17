package response

type QRResponse struct {
	Phone   string `json:"phone"`
	ChatURL string `json:"chat_url"`
	QRImage string `json:"qr_image"`
}
