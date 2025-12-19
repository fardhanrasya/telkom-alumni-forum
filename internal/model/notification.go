package model

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"` // User who receives the notification
	ActorID   uuid.UUID `gorm:"type:uuid;not null" json:"actor_id"` // User who triggered the notification
	EntityID  uuid.UUID `gorm:"type:uuid;not null" json:"entity_id"` // ID of the Post or Thread
	EntitySlug string    `gorm:"type:varchar(255)" json:"entity_slug"` // Slug of the Thread (for navigation)
	EntityType string    `gorm:"type:varchar(50);not null" json:"entity_type"` // 'thread' or 'post'
	Type      string    `gorm:"type:varchar(50);not null" json:"type"`      // 'like_thread', 'like_post', 'reply_thread', 'reply_post'
	Message   string    `gorm:"type:text" json:"message"`
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// Associations - using pointers to avoid recursion if User has Notifications
	User  *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Actor *User `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
}
