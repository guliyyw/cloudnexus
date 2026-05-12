package model

import "time"

type UserSession struct {
	ID           uint64    `json:"id,string" gorm:"primaryKey"`
	UserID       uint64    `json:"user_id,string" gorm:"not null;index"`
	JTI          string    `json:"jti" gorm:"uniqueIndex;not null;size:36"`
	UserAgent    string    `json:"user_agent" gorm:"size:500"`
	IPAddress    string    `json:"ip_address" gorm:"size:45"`
	LoginAt      time.Time `json:"login_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	ExpiresAt    time.Time `json:"expires_at" gorm:"not null"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
}
