package dto

import "github.com/google/uuid"

type ReactionToggleRequest struct {
	ReferenceID   uuid.UUID `json:"reference_id" binding:"required"`
	ReferenceType string    `json:"reference_type" binding:"required,oneof=thread post menfess"`
	Emoji         string    `json:"emoji" binding:"required"`
}

type ReactionsResponse struct {
	Counts      map[string]int64 `json:"counts"`
	UserReacted []string         `json:"user_reacted"` // List of emojis the user has reacted with
}
