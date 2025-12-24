package repository

import (
	"time"

	"anoa.com/telkomalumiforum/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LeaderboardRepository interface {
	CreatePointLog(log *model.PointLog) error
	UpdateUserStats(userID uuid.UUID, points int) error
	GetDailyThreadCount(userID uuid.UUID, date time.Time) (int64, error)
	GetTopUsers(limit int, timeframe string) ([]model.UserStats, error)
	GetUserStatsByUserID(userID uuid.UUID) (*model.UserStats, error)
}

type leaderboardRepository struct {
	db *gorm.DB
}

func NewLeaderboardRepository(db *gorm.DB) LeaderboardRepository {
	return &leaderboardRepository{db: db}
}

func (r *leaderboardRepository) CreatePointLog(log *model.PointLog) error {
	return r.db.Create(log).Error
}

func (r *leaderboardRepository) UpdateUserStats(userID uuid.UUID, points int) error {
	// Using GORM OnConflict for Upsert
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"total_score_all_time": gorm.Expr("user_stats.total_score_all_time + ?", points),
			"total_score_monthly":  gorm.Expr("user_stats.total_score_monthly + ?", points),
			"total_score_weekly":   gorm.Expr("user_stats.total_score_weekly + ?", points),
			"last_updated_at":      gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(&model.UserStats{
		UserID:            userID,
		TotalScoreAllTime: points,
		TotalScoreMonthly: points,
		TotalScoreWeekly:  points,
	}).Error
}

func (r *leaderboardRepository) GetDailyThreadCount(userID uuid.UUID, date time.Time) (int64, error) {
	var count int64
	// Filter by user_id, action_type='create_thread', and created_at on the same day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := r.db.Model(&model.PointLog{}).
		Where("user_id = ? AND action_type = ? AND created_at >= ? AND created_at < ?", userID, "create_thread", startOfDay, endOfDay).
		Count(&count).Error
	return count, err
}

func (r *leaderboardRepository) GetTopUsers(limit int, timeframe string) ([]model.UserStats, error) {
	var stats []model.UserStats

	// Calculate weekly score start date (always needed for weekly_label)
	now := time.Now()
	weeklyStartDate := now.AddDate(0, 0, -7)

	// If all_time, use the pre-calculated stats table but also calculate weekly scores
	if timeframe == "" || timeframe == "all_time" {
		// Get top users by all-time score
		err := r.db.Preload("User").Preload("User.Role").Preload("User.Profile").
			Order("total_score_all_time DESC").Limit(limit).Find(&stats).Error
		if err != nil {
			return nil, err
		}

		// Calculate weekly scores for each user
		if len(stats) > 0 {
			var userIDs []uuid.UUID
			for _, s := range stats {
				userIDs = append(userIDs, s.UserID)
			}

			// Get weekly scores from point_logs
			type WeeklyResult struct {
				UserID uuid.UUID
				Score  int
			}
			var weeklyResults []WeeklyResult
			r.db.Model(&model.PointLog{}).
				Select("user_id, SUM(points) as score").
				Where("user_id IN ? AND created_at >= ?", userIDs, weeklyStartDate).
				Group("user_id").
				Scan(&weeklyResults)

			// Map weekly scores
			weeklyMap := make(map[uuid.UUID]int)
			for _, wr := range weeklyResults {
				weeklyMap[wr.UserID] = wr.Score
			}

			// Update stats with weekly scores
			for i := range stats {
				if ws, ok := weeklyMap[stats[i].UserID]; ok {
					stats[i].TotalScoreWeekly = ws
				} else {
					stats[i].TotalScoreWeekly = 0
				}
			}
		}

		return stats, nil
	}

	// For weekly/monthly timeframe: order by period score, but still fetch all-time for rank calculation
	var startDate time.Time
	switch timeframe {
	case "weekly":
		startDate = weeklyStartDate
	case "monthly":
		startDate = now.AddDate(0, -1, 0)
	}

	type Result struct {
		UserID uuid.UUID
		Score  int
	}
	var results []Result

	// Query: Select user_id, sum(points) for the period
	err := r.db.Model(&model.PointLog{}).
		Select("user_id, SUM(points) as score").
		Where("created_at >= ?", startDate).
		Group("user_id").
		Order("score DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return stats, nil
	}

	// Fetch User details and real all-time stats
	var userIDs []uuid.UUID
	for _, res := range results {
		userIDs = append(userIDs, res.UserID)
	}

	var users []model.User
	if err := r.db.Preload("Role").Preload("Profile").Find(&users, userIDs).Error; err != nil {
		return nil, err
	}

	// Fetch real all-time stats from user_stats table
	var realStats []model.UserStats
	r.db.Where("user_id IN ?", userIDs).Find(&realStats)

	// Map Users
	userMap := make(map[uuid.UUID]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Map real all-time stats
	allTimeMap := make(map[uuid.UUID]int)
	for _, rs := range realStats {
		allTimeMap[rs.UserID] = rs.TotalScoreAllTime
	}

	// Also calculate weekly scores for weekly_label
	type WeeklyResult struct {
		UserID uuid.UUID
		Score  int
	}
	var weeklyResults []WeeklyResult
	r.db.Model(&model.PointLog{}).
		Select("user_id, SUM(points) as score").
		Where("user_id IN ? AND created_at >= ?", userIDs, weeklyStartDate).
		Group("user_id").
		Scan(&weeklyResults)

	weeklyMap := make(map[uuid.UUID]int)
	for _, wr := range weeklyResults {
		weeklyMap[wr.UserID] = wr.Score
	}

	// Construct Response with REAL all-time scores
	for _, res := range results {
		s := model.UserStats{
			UserID:            res.UserID,
			User:              userMap[res.UserID],
			TotalScoreAllTime: allTimeMap[res.UserID], // Real all-time score for rank
			TotalScoreWeekly:  weeklyMap[res.UserID],  // Weekly score for weekly_label
		}

		// Set the period-specific score
		if timeframe == "weekly" {
			s.TotalScoreWeekly = res.Score
		} else {
			s.TotalScoreMonthly = res.Score
		}

		stats = append(stats, s)
	}

	return stats, nil
}

func (r *leaderboardRepository) GetUserStatsByUserID(userID uuid.UUID) (*model.UserStats, error) {
	var stats model.UserStats
	err := r.db.Where("user_id = ?", userID).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return zero stats if user has no stats yet
			return &model.UserStats{
				UserID:            userID,
				TotalScoreAllTime: 0,
				TotalScoreMonthly: 0,
				TotalScoreWeekly:  0,
			}, nil
		}
		return nil, err
	}

	// Calculate REAL weekly score from point_logs (last 7 days)
	weeklyStartDate := time.Now().AddDate(0, 0, -7)
	var weeklyScore int
	r.db.Model(&model.PointLog{}).
		Select("COALESCE(SUM(points), 0)").
		Where("user_id = ? AND created_at >= ?", userID, weeklyStartDate).
		Scan(&weeklyScore)

	// Override with real calculated weekly score
	stats.TotalScoreWeekly = weeklyScore

	return &stats, nil
}
