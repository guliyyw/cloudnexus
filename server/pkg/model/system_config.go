package model

import "time"

type SystemConfig struct {
	Key       string    `json:"key" gorm:"primaryKey;size:128"`
	Value     string    `json:"value" gorm:"not null;text"`
	UpdatedAt time.Time `json:"updated_at"`
}
