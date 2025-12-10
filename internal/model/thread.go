package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Thread struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	CategoryID  *uuid.UUID   `gorm:"type:uuid" json:"category_id"`
	Category    Category     `gorm:"constraint:OnDelete:SET NULL" json:"category"`
	UserID      uuid.UUID    `gorm:"type:uuid" json:"user_id"`
	User        User         `gorm:"constraint:OnDelete:CASCADE" json:"user"`
	Title       string       `gorm:"size:255;not null" json:"title"`
	Slug        string       `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Content     string       `gorm:"type:text;not null" json:"content"`
	Audience    string       `gorm:"size:50;not null" json:"audience"` // 'semua', 'guru', 'siswa'
	Views       int          `gorm:"default:0" json:"views"`
	Attachments []Attachment `gorm:"foreignKey:ThreadID" json:"attachments,omitempty"`
	CreatedAt   time.Time    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time    `gorm:"autoUpdateTime" json:"updated_at"`
}

func (t *Thread) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID, err = uuid.NewV7()
	}
	return
}
