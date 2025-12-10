package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	ThreadID    uuid.UUID    `gorm:"type:uuid;not null" json:"thread_id"`
	Thread      Thread       `gorm:"constraint:OnDelete:CASCADE" json:"thread,omitempty"`
	ParentID    *uuid.UUID   `gorm:"type:uuid" json:"parent_id,omitempty"`
	Parent      *Post        `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE" json:"parent,omitempty"` // For nested replies
	UserID      uuid.UUID    `gorm:"type:uuid;not null" json:"user_id"`
	User        User         `gorm:"constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Content     string       `gorm:"type:text;not null" json:"content"`
	Attachments []Attachment `gorm:"foreignKey:PostID" json:"attachments,omitempty"`
	CreatedAt   time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
}

func (p *Post) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID, err = uuid.NewV7()
	}
	return
}
