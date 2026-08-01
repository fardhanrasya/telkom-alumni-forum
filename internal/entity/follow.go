package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserFollow struct {
	FollowerID  uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"follower_id"`
	FollowingID uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"following_id"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`

	Follower  *User `gorm:"foreignKey:FollowerID;constraint:OnDelete:CASCADE" json:"follower,omitempty"`
	Following *User `gorm:"foreignKey:FollowingID;constraint:OnDelete:CASCADE" json:"following,omitempty"`
}

type ThreadFeedView struct {
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id"`
	ThreadID uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"thread_id"`
	ViewedAt time.Time `gorm:"autoCreateTime" json:"viewed_at"`

	User   *User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Thread *Thread `gorm:"foreignKey:ThreadID;constraint:OnDelete:CASCADE" json:"thread,omitempty"`
}
