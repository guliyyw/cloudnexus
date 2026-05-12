package model

import "time"

type OAuthBinding struct {
	ID           uint64     `json:"id,string" gorm:"primaryKey"`
	UserID       uint64     `json:"user_id,string" gorm:"not null;index"`
	Provider     string     `json:"provider" gorm:"not null;size:20"`
	OpenID       string     `json:"open_id" gorm:"not null;size:255;uniqueIndex:idx_provider_open"`
	AccessToken  string     `json:"-" gorm:"type:text"`
	RefreshToken string     `json:"-" gorm:"type:text"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
