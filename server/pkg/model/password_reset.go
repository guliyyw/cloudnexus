package model

import "time"

type PasswordResetToken struct {
	ID        uint64    `json:"id,string" gorm:"primaryKey"`
	UserID    uint64    `json:"user_id,string" gorm:"not null"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null;size:64"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}
