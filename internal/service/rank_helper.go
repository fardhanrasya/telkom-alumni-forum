package service

import "math"

// GamificationStatus represents the complete gamification status of a user.
// RankName is ALWAYS based on all-time points (permanent achievement).
// WeeklyLabel provides context for recent activity (spotlight/trending).
type GamificationStatus struct {
	// All-Time Rank (permanent - never demotes)
	RankName      string  `json:"rank_name"`      // Current rank: Pendatang, Warga, Aktivis, Tokoh, Sepuh, Legenda
	NextRank      string  `json:"next_rank"`      // Next rank to achieve, or "Max Level"
	CurrentPoints int     `json:"current_points"` // All-time total points
	TargetPoints  int     `json:"target_points"`  // Points needed for next rank
	Progress      float64 `json:"progress"`       // Progress percentage to next rank (0-100)

	// Weekly Activity Context (spotlight indicator)
	WeeklyPoints int    `json:"weekly_points"` // Points earned in last 7 days
	WeeklyLabel  string `json:"weekly_label"`  // Activity label or empty string
}

// Rank thresholds (All-Time)
// These define the permanent ranks based on total accumulated points.
const (
	PointsLegenda   = 20000 // 🏆 Legenda - Hall of Fame
	PointsSepuh     = 8000  // 🎖️ Sepuh - Veteran
	PointsTokoh     = 3000  // ⭐ Tokoh - Notable figure
	PointsAktivis   = 600   // 📣 Aktivis - Active contributor
	PointsWarga     = 100   // 👤 Warga - Regular member
	PointsPendatang = 0     // 🆕 Pendatang - Newcomer
)

// Weekly activity 	thresholds
// These define the activity labels based on points earned in the last 7 days.
const (
	WeeklyOnFire   = 100 // 🔥 On Fire! - Very active this week
	WeeklyTrending = 50  // ⚡ Trending - Above average activity
	WeeklyActive   = 20  // 📈 Active - Steady contributor
)

// GetGamificationStatus calculates the gamification status based on all-time points only.
// Use this when weekly data is not available.
func GetGamificationStatus(allTimePoints int) GamificationStatus {
	return GetGamificationStatusWithWeekly(allTimePoints, 0)
}

// GetGamificationStatusWithWeekly calculates complete gamification status.
// - allTimePoints: Total accumulated points (for permanent rank)
// - weeklyPoints: Points from last 7 days (for activity label)
func GetGamificationStatusWithWeekly(allTimePoints, weeklyPoints int) GamificationStatus {
	var status GamificationStatus
	status.CurrentPoints = allTimePoints
	status.WeeklyPoints = weeklyPoints

	// 1. Calculate Main Rank (Always based on All-Time points)
	switch {
	case allTimePoints >= PointsLegenda:
		status.RankName = "Legenda"
		status.NextRank = "Max Level"
		status.TargetPoints = PointsLegenda
		status.Progress = 100

	case allTimePoints >= PointsSepuh:
		status.RankName = "Sepuh"
		status.NextRank = "Legenda"
		status.TargetPoints = PointsLegenda
		status.Progress = (float64(allTimePoints) / float64(PointsLegenda)) * 100

	case allTimePoints >= PointsTokoh:
		status.RankName = "Tokoh"
		status.NextRank = "Sepuh"
		status.TargetPoints = PointsSepuh
		status.Progress = (float64(allTimePoints) / float64(PointsSepuh)) * 100

	case allTimePoints >= PointsAktivis:
		status.RankName = "Aktivis"
		status.NextRank = "Tokoh"
		status.TargetPoints = PointsTokoh
		status.Progress = (float64(allTimePoints) / float64(PointsTokoh)) * 100

	case allTimePoints >= PointsWarga:
		status.RankName = "Warga"
		status.NextRank = "Aktivis"
		status.TargetPoints = PointsAktivis
		status.Progress = (float64(allTimePoints) / float64(PointsAktivis)) * 100

	default:
		status.RankName = "Pendatang"
		status.NextRank = "Warga"
		status.TargetPoints = PointsWarga
		if allTimePoints == 0 {
			status.Progress = 0
		} else {
			status.Progress = (float64(allTimePoints) / float64(PointsWarga)) * 100
		}
	}

	// 2. Calculate Weekly Activity Label
	switch {
	case weeklyPoints >= WeeklyOnFire:
		status.WeeklyLabel = "🔥 On Fire!"
	case weeklyPoints >= WeeklyTrending:
		status.WeeklyLabel = "⚡ Trending"
	case weeklyPoints >= WeeklyActive:
		status.WeeklyLabel = "📈 Active"
	default:
		status.WeeklyLabel = ""
	}

	// Round progress to 2 decimal places
	status.Progress = math.Round(status.Progress*100) / 100

	return status
}
