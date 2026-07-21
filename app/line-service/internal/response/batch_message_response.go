package response

type SendBatchMessageResponse struct {
	Total              int      `json:"total"`
	Sent               int      `json:"sent"`
	Failed             int      `json:"failed"`
	ChunkCount         int      `json:"chunk_count"`
	FailedRecipientIDs []string `json:"failed_recipient_ids,omitempty"`
}
