package dto

import "github.com/google/uuid"

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description"`
}

type CategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
}
