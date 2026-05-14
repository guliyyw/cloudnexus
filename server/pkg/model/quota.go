package model

import "time"

type QuotaTier struct {
	BaseModel
	Name         string `json:"name" gorm:"uniqueIndex;not null;size:64"`
	StorageLimit  int64  `json:"storage_limit" gorm:"not null"`
	Description  string `json:"description" gorm:"size:256"`
}

type UserQuota struct {
	UserID       uint64    `json:"user_id,string" gorm:"primaryKey"`
	StorageUsed  int64     `json:"storage_used" gorm:"not null;default:0"`
	StorageLimit *int64    `json:"storage_limit" gorm:"default:null"`
	TierID       *uint64   `json:"tier_id,string" gorm:"default:null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GetEffectiveLimit returns the effective storage limit for this quota.
// If StorageLimit is set (custom override), use it; otherwise look up from tier.
func (q *UserQuota) GetEffectiveLimit(tiers map[uint64]int64) int64 {
	if q.StorageLimit != nil && *q.StorageLimit > 0 {
		return *q.StorageLimit
	}
	if q.TierID != nil {
		if limit, ok := tiers[*q.TierID]; ok {
			return limit
		}
	}
	return 1073741824 // default 1GB
}
