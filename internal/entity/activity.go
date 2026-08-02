package entity

import "github.com/google/uuid"

// LoginStreak tracks a user's consecutive-day activity streak. Updated by
// POST /activity/heartbeat, called once per FE session. LastActiveDate is a
// WIB calendar date ("2006-01-02") — the same WIB-midnight rollover used by
// daily missions — not UTC.
type LoginStreak struct {
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User           *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	LastActiveDate string    `gorm:"size:10;not null" json:"last_active_date"`
	CurrentStreak  int       `gorm:"not null;default:0" json:"current_streak"`
}
