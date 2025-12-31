package dto

import "github.com/google/uuid"

type AuthorResponse struct {
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatar_url"`
}

type CategoryFilter struct {
	Search string `form:"search"`
}

type ThreadFilter struct {
	CategoryID string `form:"category_id"`
	Search     string `form:"search"`
	Audience   string `form:"audience"`
	SortBy     string `form:"sort_by"` // "newest", "popular"
	Page       int    `form:"page" binding:"min=1"`
	Limit      int    `form:"limit" binding:"min=1,max=20"`
}

type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	TotalItems  int64 `json:"total_items"`
	Limit       int   `json:"limit"`
}

type PaginatedThreadResponse struct {
	Data []ThreadResponse `json:"data"`
	Meta PaginationMeta   `json:"meta"`
}

type PaginatedCategoryResponse struct {
	Data []CategoryResponse `json:"data"`
	Meta PaginationMeta     `json:"meta"`
}

type DeleteCategoryRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

type GetCategoryRequest struct {
	ID uuid.UUID `uri:"id" binding:"required,uuid"`
}

type CategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
}

type ThreadResponse struct {
	ID           uuid.UUID            `json:"id"`
	Title        string               `json:"title"`
	Slug         string               `json:"slug"`
	Content      string               `json:"content"`
	CategoryName string               `json:"category_name"`
	Author       AuthorResponse       `json:"author"`
	CreatedAt    string               `json:"created_at"`
	UpdatedAt    string               `json:"updated_at"`
	ReplyCount   int64                `json:"reply_count"`
	Views        int                  `json:"views"`
	ImageURL     *string              `json:"image_url"`
	Audience     string               `json:"audience"`
	Attachments  []AttachmentResponse `json:"attachments"`
	Reactions    ReactionsResponse    `json:"reactions"`
}

type AttachmentResponse struct {
	ID       uint   `json:"id"`
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
}

type ReactionsResponse struct {
	Counts      map[string]int64 `json:"counts"`
	UserReacted *string          `json:"user_reacted"`
}

type GamificationStatus struct {
	RankName      string `json:"rank_name"`
	NextRank      string `json:"next_rank"`
	CurrentPoints int    `json:"current_points"`
	TargetPoints  int     `json:"target_points"`
	Progress      float64 `json:"progress"` // Percentage
	WeeklyPoints  int     `json:"weekly_points"`
	WeeklyLabel   string `json:"weekly_label"`
}

type AvatarFile struct {
	Reader   interface {
		Read(p []byte) (n int, err error)
	}
	FileName string
}
