package model

import "time"

type DockerNode struct {
	BaseModel
	Name               string     `json:"name" gorm:"uniqueIndex;not null;size:64"`
	Host               string     `json:"host" gorm:"not null;size:255"`
	Port               int        `json:"port" gorm:"default:2376"`
	TLSCert            string     `json:"-" gorm:"type:text"`
	TLSKey             string     `json:"-" gorm:"type:text"`
	CACert             string     `json:"-" gorm:"type:text"`
	Status             string     `json:"status" gorm:"default:offline;size:16"`
	NodeType           string     `json:"node_type" gorm:"default:service;size:16"`
	Service            string     `json:"service" gorm:"size:32"`
	FirstSeenAt        *time.Time `json:"first_seen_at"`
	TotalOnlineSeconds int64      `json:"total_online_seconds" gorm:"default:0"`
	OfflineSince       *time.Time `json:"offline_since"`
	ContainerName      string     `json:"container_name" gorm:"size:128"`
	Version            string     `json:"version" gorm:"size:32"`
	LastHeartbeat      *time.Time `json:"last_heartbeat"`
}
