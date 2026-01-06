package dto

import (
	"time"

	"anoa.com/telkomalumiforum/internal/entity"
	commonDto "anoa.com/telkomalumiforum/pkg/dto"
)

// UpdateProfileInput represents the input for updating user profile
type UpdateProfileInput struct {
	Username *string `json:"username" form:"username"`
	Password *string `json:"password" form:"password"`
	Bio      *string `json:"bio" form:"bio"`
}

// UpdateProfileResponse is returned when updating profile or getting current user profile
type UpdateProfileResponse struct {
	User               *entity.User                 `json:"user"`
	Profile            *entity.Profile              `json:"profile"`
	GamificationStatus commonDto.GamificationStatus `json:"gamification_status"`
}

// PublicProfileResponse is returned when viewing another user's public profile
type PublicProfileResponse struct {
	Username           string                       `json:"username"`
	Role               string                       `json:"role"`
	AvatarURL          *string                      `json:"avatar_url,omitempty"`
	CreatedAt          time.Time                    `json:"created_at"`
	Angkatan           *string                      `json:"angkatan,omitempty"`
	Bio                *string                      `json:"bio,omitempty"`
	GamificationStatus commonDto.GamificationStatus `json:"gamification_status"`
}
