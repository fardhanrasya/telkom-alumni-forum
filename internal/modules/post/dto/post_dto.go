package dto

import (
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
	"github.com/google/uuid"
)

type CreatePostRequest struct {
	ThreadID      string `json:"thread_id"`
	ParentID      string `json:"parent_id"` // Optional, for nested replies
	Content       string `json:"content" binding:"required,max=5000"`
	AttachmentIDs []uint `json:"attachment_ids"`
}

type UpdatePostRequest struct {
	Content       string `json:"content" binding:"required,max=5000"`
	AttachmentIDs []uint `json:"attachment_ids"`
}

type PostResponse struct {
	ID          uuid.UUID            `json:"id"`
	ThreadID    uuid.UUID            `json:"thread_id"`
	ParentID    *uuid.UUID           `json:"parent_id,omitempty"`
	Content     string               `json:"content"`
	Author      commonDto.AuthorResponse       `json:"author"`
	Attachments []commonDto.AttachmentResponse `json:"attachments,omitempty"`
	LikesCount  int64                `json:"likes_count"` // Deprecated
	Reactions   commonDto.ReactionsResponse    `json:"reactions"`
	Replies     []*PostResponse      `json:"replies,omitempty"`
	CreatedAt   string               `json:"created_at"`

	UpdatedAt   string               `json:"updated_at"`
}

type PostFilter struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

type PaginatedPostResponse struct {
	Data []PostResponse `json:"data"`
	Meta commonDto.PaginationMeta `json:"meta"`
}
