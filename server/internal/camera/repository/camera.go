package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type CameraRepository struct {
	db *gorm.DB
}

func NewCameraRepository(db *gorm.DB) *CameraRepository {
	return &CameraRepository{db: db}
}

// --- Camera ---

func (r *CameraRepository) ListCameras(ownerID uint64, offset, limit int) ([]model.Camera, int64, error) {
	var total int64
	var cameras []model.Camera
	q := r.db.Model(&model.Camera{}).Where("owner_id = ?", ownerID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&cameras).Error; err != nil {
		return nil, 0, err
	}
	return cameras, total, nil
}

func (r *CameraRepository) FindCameraByID(id uint64) (*model.Camera, error) {
	var c model.Camera
	if err := r.db.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CameraRepository) CreateCamera(c *model.Camera) error {
	return r.db.Create(c).Error
}

func (r *CameraRepository) UpdateCamera(c *model.Camera) error {
	return r.db.Save(c).Error
}

func (r *CameraRepository) DeleteCamera(id uint64) error {
	return r.db.Delete(&model.Camera{}, "id = ?", id).Error
}

func (r *CameraRepository) UpdateCameraStatus(id uint64, status string) error {
	return r.db.Model(&model.Camera{}).Where("id = ?", id).Update("status", status).Error
}

// --- RecognitionEvent ---

func (r *CameraRepository) ListEvents(cameraID uint64, offset, limit int) ([]model.RecognitionEvent, int64, error) {
	var total int64
	var events []model.RecognitionEvent
	q := r.db.Model(&model.RecognitionEvent{}).Where("camera_id = ?", cameraID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *CameraRepository) CreateEvent(e *model.RecognitionEvent) error {
	return r.db.Create(e).Error
}
