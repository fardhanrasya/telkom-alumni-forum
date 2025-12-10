package model

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid" json:"user_id"`
	ThreadID  *uuid.UUID `gorm:"type:uuid" json:"thread_id,omitempty"`
	PostID    *uuid.UUID `gorm:"type:uuid" json:"post_id,omitempty"`
	FileURL   string     `gorm:"type:text;not null" json:"file_url"`
	FileType  string     `gorm:"size:50" json:"file_type"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
}
