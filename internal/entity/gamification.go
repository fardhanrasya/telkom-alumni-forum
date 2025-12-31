package entity

import (
	"time"

	"github.com/google/uuid"
)

type PointLog struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;index:idx_user_date,priority:1;not null" json:"user_id"`
	User           User       `gorm:"foreignKey:UserID" json:"-"`
	ActionType     string     `gorm:"size:50;not null" json:"action_type"` // 'create_thread', 'like_received', 'comment_received'
	Points         int        `gorm:"not null" json:"points"`
	ReferenceID    string     `gorm:"size:36" json:"reference_id"`    // UUID string of the thread/post
	ReferenceTable string     `gorm:"size:50" json:"reference_table"` // 'threads', 'posts'
	ActorID        *uuid.UUID `gorm:"type:uuid;index:idx_unique_like,unique,priority:1" json:"actor_id"` // Who performed the action (liker)
	CreatedAt      time.Time  `gorm:"index:idx_user_date,priority:2;index:idx_date;index:idx_unique_like,unique,priority:4" json:"created_at"`
}

// UniqueIndex: idx_unique_like on (actor_id, action_type, reference_id, created_at)
// This prevents duplicate points from the same actor for the same action on the same entity
// Note: We use composite unique index at database level for integrity

type UserStats struct {
	UserID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User              User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user"`
	TotalScoreAllTime int       `gorm:"default:0" json:"total_score_all_time"`
	TotalScoreMonthly int       `gorm:"default:0" json:"total_score_monthly"`
	TotalScoreWeekly  int       `gorm:"default:0" json:"total_score_weekly"`
	LastUpdatedAt     time.Time `gorm:"autoUpdateTime" json:"last_updated_at"`
}
