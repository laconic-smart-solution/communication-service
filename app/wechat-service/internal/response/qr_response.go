package response

type QRResponse struct {
	SceneStr string `json:"scene_str,omitempty"`
	ChatURL  string `json:"chat_url"`
	QRImage  string `json:"qr_image"`
}
