package model

import "time"

type PublicTrack struct {
	ID         uint64    `json:"id,string" gorm:"primaryKey"`
	Title      string    `json:"title" gorm:"size:300"`
	Artist     string    `json:"artist" gorm:"size:200"`
	Album      string    `json:"album" gorm:"size:200"`
	Duration   int       `json:"duration"`
	TrackNum   int       `json:"track_num"`
	StorageKey string    `json:"storage_key" gorm:"size:500"`
	MimeType   string    `json:"mime_type" gorm:"size:50"`
	FileSize   int64     `json:"file_size"`
	UploadedBy uint64    `json:"uploaded_by,string"`
	CreatedAt  time.Time `json:"created_at"`
}
