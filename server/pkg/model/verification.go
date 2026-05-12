package model

import "time"

type EmailVerification struct {
	ID        uint64    `json:"id,string" gorm:"primaryKey"`
	UserID    uint64    `json:"user_id,string" gorm:"index"`
	Email     string    `json:"email" gorm:"not null;size:255"`
	Code      string    `json:"code" gorm:"not null;size:6"`
	Type      string    `json:"type" gorm:"not null;size:20;default:register"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

type PhoneVerification struct {
	ID        uint64    `json:"id,string" gorm:"primaryKey"`
	UserID    uint64    `json:"user_id,string" gorm:"index"`
	Phone     string    `json:"phone" gorm:"not null;size:20"`
	Code      string    `json:"code" gorm:"not null;size:6"`
	Type      string    `json:"type" gorm:"not null;size:20;default:register"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}
