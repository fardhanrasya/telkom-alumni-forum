package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Reaction struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index:idx_reactions_unique,priority:1" json:"user_id"`
	User          User      `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	ReferenceID   uuid.UUID `gorm:"type:uuid;not null;index:idx_reactions_unique,priority:2;index:idx_reactions_lookup,priority:1" json:"reference_id"`
	ReferenceType string    `gorm:"size:20;not null;index:idx_reactions_unique,priority:3;index:idx_reactions_lookup,priority:2" json:"reference_type"` // 'thread', 'post', 'menfess'
	Emoji         string    `gorm:"size:10;not null;index:idx_reactions_unique,priority:4" json:"emoji"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (r *Reaction) TableName() string {
	return "reactions"
}

func (r *Reaction) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID, err = uuid.NewV7()
	}
	return
}

