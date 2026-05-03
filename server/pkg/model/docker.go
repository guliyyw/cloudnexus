package model

import "time"

type DockerNode struct {
	BaseModel
	Name          string     `json:"name" gorm:"uniqueIndex;not null;size:64"`
	Host          string     `json:"host" gorm:"not null;size:255"`
	Port          int        `json:"port" gorm:"default:2376"`
	TLSCert       string     `json:"-" gorm:"type:text"`
	TLSKey        string     `json:"-" gorm:"type:text"`
	CACert        string     `json:"-" gorm:"type:text"`
	Status        string     `json:"status" gorm:"default:offline;size:16"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`
}
