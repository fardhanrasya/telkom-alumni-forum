package dto

import (
	"time"

	"anoa.com/telkomalumiforum/pkg/dto"
)

type CreateMenfessRequest struct {
	Content string `json:"content" binding:"required,max=3000"`
}

type MenfessResponse struct {
	ID        string                `json:"id"`
	Content   string                `json:"content"`
	Reactions dto.ReactionsResponse `json:"reactions"`
	CreatedAt time.Time             `json:"created_at"`
}
