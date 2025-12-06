package dto

import (
	"mime/multipart"

	"github.com/google/uuid"
)

type CreateThreadRequest struct {
	CategoryID  string                  `form:"category_id" binding:"required,uuid"`
	Title       string                  `form:"title" binding:"required,max=255"`
	Content     string                  `form:"content" binding:"required"`
	Audience    string                  `form:"audience" binding:"required,oneof=semua guru siswa"`
	Attachments []*multipart.FileHeader `form:"attachments"`
}

type ThreadResponse struct {
	ID           uuid.UUID            `json:"id"`
	CategoryName string               `json:"category_name"`
	Title        string               `json:"title"`
	Slug         string               `json:"slug"`
	Content      string               `json:"content"`
	Audience     string               `json:"audience"`
	Views        int                  `json:"views"`
	Author       string               `json:"author"`
	Attachments  []AttachmentResponse `json:"attachments,omitempty"`
	CreatedAt    string               `json:"created_at"`
}

type AttachmentResponse struct {
	ID       uint   `json:"id"`
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
}
