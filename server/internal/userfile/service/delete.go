package service

import (
	"time"

	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type DeleteService struct {
	db *gorm.DB
}

func NewDeleteService(db *gorm.DB) *DeleteService {
	return &DeleteService{db: db}
}

func (s *DeleteService) RequestDelete(userID uint64) error {
	now := time.Now()
	return s.db.Model(&model.User{}).Where("id = ?", userID).
		Update("delete_requested_at", now).Error
}

func (s *DeleteService) CancelDelete(userID uint64) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).
		Update("delete_requested_at", nil).Error
}

func (s *DeleteService) ConfirmDelete(userID uint64) error {
	now := time.Now()
	return s.db.Model(&model.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"deleted_at":          now,
			"delete_requested_at": nil,
		}).Error
}

func (s *DeleteService) CleanupDeletedUsers() error {
	threshold := time.Now().Add(-30 * 24 * time.Hour)
	result := s.db.Where("deleted_at IS NOT NULL AND deleted_at < ?", threshold).Delete(&model.User{})
	if result.Error != nil {
		return apperrors.NewAppError(500, "清理已删除用户失败", result.Error)
	}
	return nil
}
