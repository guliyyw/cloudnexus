package service

import (
	"runtime"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type HealthHistoryService struct {
	db *gorm.DB
}

func NewHealthHistoryService(db *gorm.DB) *HealthHistoryService {
	return &HealthHistoryService{db: db}
}

// SaveSnapshot stores a dashboard status snapshot as JSON.
func (s *HealthHistoryService) SaveSnapshot(statusData string) error {
	snap := model.DashboardHealthSnapshot{
		Timestamp:  time.Now(),
		StatusData: statusData,
		Type:       "health",
	}
	return s.db.Create(&snap).Error
}

// SaveResourceMetric stores a single resource metric point.
func (s *HealthHistoryService) SaveResourceMetric(service string, cpuPercent float64, memoryUsed, memoryTotal int64) error {
	metric := model.ResourceMetric{
		Timestamp:   time.Now(),
		Service:     service,
		CPUPercent:  cpuPercent,
		MemoryUsed:  memoryUsed,
		MemoryTotal: memoryTotal,
	}
	return s.db.Create(&metric).Error
}

// HealthSnapshot represents a lightweight snapshot for the API response.
type HealthSnapshot struct {
	Timestamp time.Time `json:"timestamp"`
	Data      string    `json:"data"`
}

// GetHealthHistory returns health snapshots within the given time range.
func (s *HealthHistoryService) GetHealthHistory(since time.Time) ([]HealthSnapshot, error) {
	var snaps []model.DashboardHealthSnapshot
	err := s.db.Where("type = ? AND timestamp >= ?", "health", since).
		Order("timestamp ASC").Find(&snaps).Error
	if err != nil {
		return nil, err
	}
	result := make([]HealthSnapshot, 0, len(snaps))
	for _, snap := range snaps {
		result = append(result, HealthSnapshot{
			Timestamp: snap.Timestamp,
			Data:      snap.StatusData,
		})
	}
	return result, nil
}

// ResourceHistoryResponse contains resource metrics grouped by service.
type ResourceHistoryResponse struct {
	Services map[string][]ResourcePoint `json:"services"`
}

// ResourcePoint is a single resource data point.
type ResourcePoint struct {
	Timestamp   string  `json:"timestamp"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryUsed  int64   `json:"memory_used"`
	MemoryTotal int64   `json:"memory_total"`
}

// GetResourceHistory returns resource metrics within the given time range,
// optionally filtered by service.
func (s *HealthHistoryService) GetResourceHistory(since time.Time, service string) (*ResourceHistoryResponse, error) {
	query := s.db.Where("timestamp >= ?", since)
	if service != "" && service != "all" {
		query = query.Where("service = ?", service)
	}

	var metrics []model.ResourceMetric
	if err := query.Order("timestamp ASC").Find(&metrics).Error; err != nil {
		return nil, err
	}

	services := make(map[string][]ResourcePoint)
	for _, m := range metrics {
		svc := m.Service
		if svc == "" {
			svc = "unknown"
		}
		services[svc] = append(services[svc], ResourcePoint{
			Timestamp:   m.Timestamp.Format(time.RFC3339),
			CPUPercent:  m.CPUPercent,
			MemoryUsed:  m.MemoryUsed,
			MemoryTotal: m.MemoryTotal,
		})
	}
	return &ResourceHistoryResponse{Services: services}, nil
}

// CollectResourceMetrics samples Go runtime memory every minute and stores
// it as a ResourceMetric for the local service.
func CollectResourceMetrics(svc *HealthHistoryService) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		svc.SaveResourceMetric("user-file-svc", 0, int64(m.Alloc), int64(m.Sys))
	}
}

// CleanOldSnapshots deletes snapshots older than the retention duration.
func (s *HealthHistoryService) CleanOldSnapshots(retention time.Duration) error {
	cutoff := time.Now().Add(-retention)
	if err := s.db.Where("timestamp < ?", cutoff).Delete(&model.DashboardHealthSnapshot{}).Error; err != nil {
		return err
	}
	return s.db.Where("timestamp < ?", cutoff).Delete(&model.ResourceMetric{}).Error
}
