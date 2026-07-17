package response

type QRResponse struct {
	BotUsername string `json:"bot_username"`
	BotURL      string `json:"bot_url"`
	QRImage     string `json:"qr_image"`
}
