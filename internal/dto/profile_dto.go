package dto

import (
	"time"

	"anoa.com/telkomalumiforum/internal/model"
)

// UpdateProfileInput represents the input for updating user profile
type UpdateProfileInput struct {
	Username *string `json:"username" form:"username"`
	Password *string `json:"password" form:"password"`
	Bio      *string `json:"bio" form:"bio"`
}

// GamificationStatus represents the gamification status in API responses.
// RankName is ALWAYS based on all-time points (permanent achievement).
// WeeklyLabel provides context for recent activity (spotlight/trending).
type GamificationStatus struct {
	// All-Time Rank (permanent - never demotes based on timeframe)
	RankName      string  `json:"rank_name"`      // Pendatang, Warga, Aktivis, Tokoh, Sepuh, Legenda
	NextRank      string  `json:"next_rank"`      // Next rank to achieve
	CurrentPoints int     `json:"current_points"` // All-time total points
	TargetPoints  int     `json:"target_points"`  // Points needed for next rank
	Progress      float64 `json:"progress"`       // Progress percentage (0-100)

	// Weekly Activity Context (spotlight indicator)
	WeeklyPoints int    `json:"weekly_points"` // Points earned in last 7 days
	WeeklyLabel  string `json:"weekly_label"`  // "🔥 On Fire!", "⚡ Trending", "📈 Active", or ""
}

// UpdateProfileResponse is returned when updating profile or getting current user profile
type UpdateProfileResponse struct {
	User               *model.User        `json:"user"`
	Profile            *model.Profile     `json:"profile"`
	GamificationStatus GamificationStatus `json:"gamification_status"`
}

// PublicProfileResponse is returned when viewing another user's public profile
type PublicProfileResponse struct {
	Username           string             `json:"username"`
	Role               string             `json:"role"`
	AvatarURL          *string            `json:"avatar_url,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	ClassGrade         *string            `json:"class_grade,omitempty"`
	Bio                *string            `json:"bio,omitempty"`
	GamificationStatus GamificationStatus `json:"gamification_status"`
}
