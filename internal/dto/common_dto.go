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
