package response

type QRResponse struct {
	URI     string `json:"uri"`
	BotURL  string `json:"bot_url"`
	QRImage string `json:"qr_image"`
}
