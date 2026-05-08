package model

import "time"

type FileVersion struct {
	ID         uint64    `json:"id,string" gorm:"primaryKey"`
	FileID     uint64    `json:"file_id,string" gorm:"not null;index"`
	VersionNum int       `json:"version_num" gorm:"not null"`
	StorageKey string    `json:"storage_key" gorm:"not null;size:512"`
	Size       int64     `json:"size"`
	SHA256     string    `json:"sha256" gorm:"size:64"`
	Message    string    `json:"message" gorm:"size:256"`
	CreatedAt  time.Time `json:"created_at"`
}
