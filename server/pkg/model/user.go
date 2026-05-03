package model

import "time"

type User struct {
	BaseModel
	Username string `json:"username" gorm:"uniqueIndex;not null;size:64"`
	Email    string `json:"email" gorm:"uniqueIndex;not null;size:255"`
	Password string `json:"-" gorm:"not null;size:255"`
	Avatar   string `json:"avatar" gorm:"size:512"`
	Status   int8   `json:"status" gorm:"default:1"`
}

type RefreshToken struct {
	ID        uint64    `json:"id" gorm:"primaryKey"`
	UserID    uint64    `json:"user_id" gorm:"not null;index"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null;size:512"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}
