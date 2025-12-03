package model

import "time"

type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email        string    `gorm:"size:100;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	RoleID       *uint     `json:"role_id"`
	Role         Role      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"role"`
	AvatarURL    *string   `gorm:"type:text" json:"avatar_url,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	Profile      *Profile  `gorm:"constraint:OnDelete:CASCADE" json:"profile,omitempty"`
}

type Profile struct {
	UserID         uint      `gorm:"primaryKey" json:"user_id"`
	FullName       string    `gorm:"size:100;not null" json:"full_name"`
	IdentityNumber *string   `gorm:"size:50" json:"identity_number,omitempty"`
	ClassGrade     *string   `gorm:"size:20" json:"class_grade,omitempty"`
	Bio            *string   `gorm:"type:text" json:"bio,omitempty"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}
