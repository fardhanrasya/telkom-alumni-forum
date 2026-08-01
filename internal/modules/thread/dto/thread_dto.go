package dto

type CreateThreadRequest struct {
	CategoryID    string `json:"category_id"`
	Title         string `json:"title"`
	Content       string `json:"content" binding:"required,max=10000"`
	Audience      string `json:"audience"`
	AttachmentIDs []uint `json:"attachment_ids"`
}

type UpdateThreadRequest struct {
	CategoryID    string `json:"category_id"`
	Title         string `json:"title"`
	Content       string `json:"content" binding:"required,max=10000"`
	Audience      string `json:"audience"`
	AttachmentIDs []uint `json:"attachment_ids"`
}
