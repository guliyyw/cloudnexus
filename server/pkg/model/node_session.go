package model

import "time"

// NodeOnlineSession records a continuous online period for a node.
type NodeOnlineSession struct {
	BaseModel
	NodeName      string     `json:"node_name" gorm:"not null;size:64;index"`
	StartTime     time.Time  `json:"start_time" gorm:"not null"`
	EndTime       *time.Time `json:"end_time"`
	Duration      int64      `json:"duration" gorm:"default:0"`
	ContainerName string     `json:"container_name" gorm:"size:128"`
	Version       string     `json:"version" gorm:"size:32"`
}
