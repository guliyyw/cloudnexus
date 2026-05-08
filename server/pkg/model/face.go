package model

import "time"

// FaceProfile represents a registered face in the user's face library.
type FaceProfile struct {
	BaseModel
	OwnerID      uint64 `json:"owner_id,string" gorm:"index"`
	Name         string `json:"name" gorm:"not null;size:64"`
	Embedding    string `json:"embedding" gorm:"type:text;not null"` // JSON array of 128 floats
	ThumbnailURL string `json:"thumbnail_url" gorm:"size:512"`
}

// FaceRecognitionEvent records a face match event from a camera stream.
type FaceRecognitionEvent struct {
	ID          uint64    `json:"id,string" gorm:"primaryKey"`
	CameraID    uint64    `json:"camera_id,string" gorm:"not null;index"`
	FaceID      *uint64   `json:"face_id,string" gorm:"index"`
	FaceName    string    `json:"face_name" gorm:"size:64"`
	Confidence  float64   `json:"confidence"`
	SnapshotURL string    `json:"snapshot_url" gorm:"size:512"`
	BboxJSON    string    `json:"bbox_json" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"index"`
}
