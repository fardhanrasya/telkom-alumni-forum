package dto

import commonDto "anoa.com/telkomalumiforum/pkg/dto"

// LeaderboardEntry represents a single user entry in the leaderboard.
// Position is the ranking in the leaderboard (1-based).
// GamificationStatus contains the user's rank and activity information.
type LeaderboardEntry struct {
	Username           string             `json:"username"`
	AvatarURL          *string            `json:"avatar_url,omitempty"`
	Role               string             `json:"role"`
	Position           int                `json:"position"` // 1-based position in leaderboard
	GamificationStatus commonDto.GamificationStatus `json:"gamification_status"`
}
