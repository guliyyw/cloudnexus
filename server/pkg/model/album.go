package model

import "time"

type Album struct {
	BaseModel
	OwnerID     uint64 `json:"owner_id,string" gorm:"index:idx_owner"`
	Name        string `json:"name" gorm:"not null;size:200"`
	Description string `json:"description" gorm:"type:text"`
	CoverFileID uint64 `json:"cover_file_id,string"`
	FileCount   int    `json:"file_count" gorm:"-"`
}

type AlbumFile struct {
	AlbumID uint64    `json:"album_id,string" gorm:"primaryKey"`
	FileID  uint64    `json:"file_id,string" gorm:"primaryKey"`
	AddedAt time.Time `json:"added_at"`
}
