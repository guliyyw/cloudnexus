package model

import "time"

// DashboardHealthSnapshot stores a point-in-time health snapshot for the dashboard.
type DashboardHealthSnapshot struct {
	ID         uint64    `json:"id,string" gorm:"primaryKey"`
	Timestamp  time.Time `json:"timestamp" gorm:"index"`
	StatusData string    `json:"status_data" gorm:"type:text"` // JSON string of DashboardStatus
	Type       string    `json:"type" gorm:"type:varchar(20);index"` // "health" | "resources"
}

// ResourceMetric stores per-minute CPU/memory sampling for each service.
type ResourceMetric struct {
	ID          uint64    `json:"id,string" gorm:"primaryKey"`
	Timestamp   time.Time `json:"timestamp" gorm:"index:idx_metric_time"`
	Service     string    `json:"service" gorm:"type:varchar(30);index:idx_metric_service_time"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemoryUsed  int64     `json:"memory_used"`
	MemoryTotal int64     `json:"memory_total"`
}
