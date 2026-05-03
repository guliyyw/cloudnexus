package model

import "time"

type File struct {
	BaseModel
	UserID        uint64 `json:"user_id" gorm:"not null;index"`
	Name          string `json:"name" gorm:"not null;size:255"`
	IsDir         bool   `json:"is_dir" gorm:"default:false"`
	ParentID      uint64 `json:"parent_id" gorm:"default:0;index"`
	Size          int64  `json:"size" gorm:"default:0"`
	MimeType      string `json:"mime_type" gorm:"size:128"`
	StorageKey    string `json:"storage_key" gorm:"size:512"`
	StorageSHA256 string `json:"storage_sha256" gorm:"size:64"`
	IsShared      bool   `json:"is_shared" gorm:"default:false"`
	DeletedAt     *time.Time
}

type FileShare struct {
	ID            uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	FileID        uint64     `json:"file_id" gorm:"not null"`
	OwnerID       uint64     `json:"owner_id" gorm:"not null"`
	ShareCode     string     `json:"share_code" gorm:"uniqueIndex;not null;size:32"`
	Password      string     `json:"-" gorm:"size:255"`
	ExpiresAt     *time.Time `json:"expires_at"`
	DownloadLimit int        `json:"download_limit" gorm:"default:0"`
	DownloadCount int        `json:"download_count" gorm:"default:0"`
	CreatedAt     time.Time  `json:"created_at"`
}
