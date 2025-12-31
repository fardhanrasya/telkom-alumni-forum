package dto

import (
	"io"

	"anoa.com/telkomalumiforum/internal/entity"
)

// AvatarFile merepresentasikan file avatar yang diupload user.
type AvatarFile struct {
	Reader   io.Reader
	FileName string
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken string         `json:"access_token"`
	TokenType   string         `json:"token_type"`
	ExpiresIn   int64          `json:"expires_in"`
	User        *entity.User    `json:"user"`
	Role        *entity.Role    `json:"role"`
	Profile     *entity.Profile `json:"profile"`
	SearchToken string         `json:"search_token,omitempty"`
}
