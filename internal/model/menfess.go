package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Menfess struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `gorm:"type:timestamp" json:"created_at"` // Fuzzy timestamp
	// No UserID, No UpdatedAt
}

func (m *Menfess) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID, err = uuid.NewV7()
	}
	return
}
