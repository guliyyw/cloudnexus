package model

import "time"

type MusicLike struct {
	UserID    uint64    `json:"user_id,string" gorm:"primaryKey"`
	TrackID   uint64    `json:"track_id,string" gorm:"primaryKey"`
	Source    string    `json:"source" gorm:"primaryKey;size:10"`
	CreatedAt time.Time `json:"created_at"`
}

type MusicRecentPlay struct {
	UserID   uint64    `json:"user_id,string" gorm:"primaryKey"`
	TrackID  uint64    `json:"track_id,string" gorm:"primaryKey"`
	Source   string    `json:"source" gorm:"primaryKey;size:10"`
	PlayedAt time.Time `json:"played_at"`
}
