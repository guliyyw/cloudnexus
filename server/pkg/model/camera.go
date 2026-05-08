package model

import "time"

// Camera represents a configured RTSP/RTMP/ONVIF camera.
type Camera struct {
	BaseModel
	OwnerID    uint64     `json:"owner_id,string" gorm:"index"`
	Name       string     `json:"name" gorm:"not null;size:128"`
	StreamURL  string     `json:"stream_url" gorm:"not null;size:512"`
	Protocol   string     `json:"protocol" gorm:"default:rtsp;size:16"`
	Status     string     `json:"status" gorm:"default:offline;size:16"`
	LastSeenAt *time.Time `json:"last_seen_at"`
}

// RecognitionEvent represents an AI-detected object in a camera frame.
type RecognitionEvent struct {
	ID          uint64    `json:"id,string" gorm:"primaryKey"`
	CameraID    uint64    `json:"camera_id,string" gorm:"not null;index"`
	EventType   string    `json:"event_type" gorm:"not null;size:32"`
	Confidence  float64   `json:"confidence"`
	SnapshotURL string    `json:"snapshot_url" gorm:"size:512"`
	Metadata    string    `json:"metadata" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at" gorm:"index"`
}
