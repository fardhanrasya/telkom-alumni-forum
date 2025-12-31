package dto

type UploadAttachmentResponse struct {
	ID       uint   `json:"id"`
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
}

type AttachmentResponse struct {
	ID       uint   `json:"id"`
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
}
