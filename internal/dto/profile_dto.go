package dto

import (
	"time"

	"anoa.com/telkomalumiforum/internal/model"
)

type UpdateProfileInput struct {
	Username *string `json:"username" form:"username"`
	Password *string `json:"password" form:"password"`
	Bio      *string `json:"bio" form:"bio"`
}

type UpdateProfileResponse struct {
	User    *model.User    `json:"user"`
	Profile *model.Profile `json:"profile"`
}

type PublicProfileResponse struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Angkatan  *string   `json:"angkatan,omitempty"`
	Bio       *string   `json:"bio,omitempty"`
}
