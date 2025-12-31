package dto

type CreateThreadRequest struct {
	CategoryID    string `json:"category_id" binding:"required,uuid"`
	Title         string `json:"title" binding:"required,max=120"`
	Content       string `json:"content" binding:"required,max=10000"`
	Audience      string `json:"audience" binding:"required,oneof=semua guru siswa"`
	AttachmentIDs []uint `json:"attachment_ids"`
}

type UpdateThreadRequest struct {
	CategoryID    string `json:"category_id" binding:"required,uuid"`
	Title         string `json:"title" binding:"required,max=120"`
	Content       string `json:"content" binding:"required,max=10000"`
	Audience      string `json:"audience" binding:"required,oneof=semua guru siswa"`
	AttachmentIDs []uint `json:"attachment_ids"`
}


