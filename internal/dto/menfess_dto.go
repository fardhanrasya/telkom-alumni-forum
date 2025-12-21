package dto

import "time"

type CreateMenfessRequest struct {
	Content string `json:"content" binding:"required,max=3000"`
}

type MenfessResponse struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
