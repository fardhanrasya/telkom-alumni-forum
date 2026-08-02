package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Wallet holds a user's Tel-Credits ("TC") balance. Coin is a ledger separate
// from the point/rank system and never affects rank.
type Wallet struct {
	UserID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User    *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Balance int       `gorm:"not null;default:0" json:"balance"`
}

// CoinTransaction is an append-only ledger entry — never edited or deleted.
// Mistakes are reversed with a correction entry (negative Amount), not a delete.
type CoinTransaction struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	Amount       int       `gorm:"not null" json:"amount"` // + grant, - purchase
	SourceType   string    `gorm:"size:50;index:idx_coin_tx_idempotency,unique,priority:1" json:"source_type"`
	SourceRefID  string    `gorm:"size:100;index:idx_coin_tx_idempotency,unique,priority:2" json:"source_ref_id"`
	BalanceAfter int       `gorm:"not null" json:"balance_after"`
	CreatedAt    time.Time `json:"created_at"`
}

// Cosmetic is a catalog entry. Payload shape depends on RenderType:
// "css" -> {preset_key}, "image" -> {animated_url, static_url}.
type Cosmetic struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Slot       string         `gorm:"size:20;not null" json:"slot"`        // "avatar_border" | "thread_bg" | "profile_bg"
	SubType    string         `gorm:"size:20" json:"sub_type"`             // avatar_border only: "ring" | "decoration"
	RenderType string         `gorm:"size:10;not null" json:"render_type"` // "css" | "image"
	Name       string         `gorm:"size:100;not null" json:"name"`
	Payload    datatypes.JSON `json:"payload"`
	Price      int            `gorm:"not null" json:"price"`
	MinRank    string         `gorm:"size:20" json:"min_rank"`                      // empty = no gate
	Status     string         `gorm:"size:20;not null;default:draft" json:"status"` // "draft" | "published" | "retired"
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// CosmeticOwnership records a permanent purchase. Composite PK enforces
// one-purchase-per-user-per-cosmetic, same idea as idx_unique_like.
type CosmeticOwnership struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	CosmeticID  uint      `gorm:"primaryKey" json:"cosmetic_id"`
	User        *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Cosmetic    *Cosmetic `gorm:"foreignKey:CosmeticID;constraint:OnDelete:CASCADE" json:"cosmetic,omitempty"`
	PurchasedAt time.Time `gorm:"autoCreateTime" json:"purchased_at"`
}

// UserEquip is one row per user with three nullable slot FKs, not a generic
// per-slot table — a user can only ever have one active item per slot.
type UserEquip struct {
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User           *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	AvatarBorderID *uint     `json:"avatar_border_id"`
	AvatarBorder   *Cosmetic `gorm:"foreignKey:AvatarBorderID" json:"avatar_border,omitempty"`
	ThreadBgID     *uint     `json:"thread_bg_id"`
	ThreadBg       *Cosmetic `gorm:"foreignKey:ThreadBgID" json:"thread_bg,omitempty"`
	ProfileBgID    *uint     `json:"profile_bg_id"`
	ProfileBg      *Cosmetic `gorm:"foreignKey:ProfileBgID" json:"profile_bg,omitempty"`
}

// MissionDefinition is data-driven: action type is a hardcoded enum, but
// target/reward/is_active are editable by admins without a deploy.
type MissionDefinition struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ActionType string    `gorm:"size:30;not null" json:"action_type"` // create_thread | like_received | comment_received | follow | view_thread | login_streak
	Kind       string    `gorm:"size:15;not null" json:"kind"`        // "daily" | "achievement"
	Name       string    `gorm:"size:100;not null" json:"name"`
	Target     int       `gorm:"not null" json:"target"`
	Reward     int       `gorm:"not null" json:"reward"`
	IsActive   bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MissionProgress tracks one user's progress on one mission for one period.
// Period is a WIB calendar date ("2006-01-02") for daily missions, or the
// literal string "lifetime" for achievements. Unresolved daily progress
// expires (no carry-over) simply by the period no longer matching "today".
type MissionProgress struct {
	ID        uint               `gorm:"primaryKey" json:"id"`
	UserID    uuid.UUID          `gorm:"type:uuid;index:idx_unique_progress,unique,priority:1" json:"user_id"`
	MissionID uint               `gorm:"index:idx_unique_progress,unique,priority:2" json:"mission_id"`
	Mission   *MissionDefinition `gorm:"foreignKey:MissionID;constraint:OnDelete:CASCADE" json:"mission,omitempty"`
	Period    string             `gorm:"size:20;index:idx_unique_progress,unique,priority:3" json:"period"`
	Progress  int                `gorm:"not null;default:0" json:"progress"`
	ClaimedAt *time.Time         `json:"claimed_at"`
}
