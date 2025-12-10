package dto

import "github.com/google/uuid"

type CreatePostRequest struct {
	ThreadID      string `json:"thread_id"`
	ParentID      string `json:"parent_id"` // Optional, for nested replies
	Content       string `json:"content" binding:"required"`
	AttachmentIDs []uint `json:"attachment_ids"`
}

type UpdatePostRequest struct {
	Content       string `json:"content" binding:"required"`
	AttachmentIDs []uint `json:"attachment_ids"`
}

type PostResponse struct {
	ID          uuid.UUID            `json:"id"`
	ThreadID    uuid.UUID            `json:"thread_id"`
	ParentID    *uuid.UUID           `json:"parent_id,omitempty"`
	Content     string               `json:"content"`
	Author      string               `json:"author"`
	Attachments []AttachmentResponse `json:"attachments,omitempty"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

type PostFilter struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

type PaginatedPostResponse struct {
	Data []PostResponse `json:"data"`
	Meta PaginationMeta `json:"meta"`
}
