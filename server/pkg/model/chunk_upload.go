package model

type ChunkUpload struct {
	BaseModel
	UserID         uint64  `json:"user_id,string" gorm:"not null;index"`
	UploadID       string  `json:"upload_id" gorm:"uniqueIndex;not null;size:36"`
	FileName       string  `json:"file_name" gorm:"not null;size:255"`
	FileSize       int64   `json:"file_size" gorm:"not null"`
	ChunkSize      int     `json:"chunk_size" gorm:"not null;default:10485760"`
	MimeType       string  `json:"mime_type" gorm:"size:128"`
	ParentID       uint64  `json:"parent_id,string" gorm:"default:0"`
	TotalChunks    int     `json:"total_chunks" gorm:"not null"`
	Completed      Int32Array `json:"completed" gorm:"type:integer[];default:'{}'"`
	Status         string  `json:"status" gorm:"not null;size:20;default:uploading"`
	VersionMessage string  `json:"version_message" gorm:"size:256"`
}
