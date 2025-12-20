package dto

import (
	"github.com/google/uuid"
)

type CreateThreadRequest struct {
	CategoryID    string `json:"category_id" binding:"required,uuid"`
	Title         string `json:"title" binding:"required,max=255"`
	Content       string `json:"content" binding:"required"`
	Audience      string `json:"audience" binding:"required,oneof=semua guru siswa"`
	AttachmentIDs []uint `json:"attachment_ids"`
}

type UpdateThreadRequest struct {
	CategoryID    string `json:"category_id" binding:"required,uuid"`
	Title         string `json:"title" binding:"required,max=255"`
	Content       string `json:"content" binding:"required"`
	Audience      string `json:"audience" binding:"required,oneof=semua guru siswa"`
	AttachmentIDs []uint `json:"attachment_ids"`
}

type ThreadResponse struct {
	ID           uuid.UUID            `json:"id"`
	CategoryName string               `json:"category_name"`
	Title        string               `json:"title"`
	Slug         string               `json:"slug"`
	Content      string               `json:"content"`
	Audience     string               `json:"audience"`
	Views        int                  `json:"views"`
	Author       AuthorResponse       `json:"author"`
	Attachments  []AttachmentResponse `json:"attachments,omitempty"`
	LikesCount   int64                `json:"likes_count"`
	CreatedAt    string               `json:"created_at"`
}
