package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChunkRepository struct {
	db *gorm.DB
}

func NewChunkRepository(db *gorm.DB) *ChunkRepository {
	return &ChunkRepository{db: db}
}

func (r *ChunkRepository) Create(chunk *model.ChunkUpload) error {
	return r.db.Create(chunk).Error
}

func (r *ChunkRepository) FindByUploadID(uploadID string) (*model.ChunkUpload, error) {
	var c model.ChunkUpload
	err := r.db.Where("upload_id = ?", uploadID).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ChunkRepository) AddCompletedChunk(uploadID string, chunkIndex int32) error {
	return r.db.Model(&model.ChunkUpload{}).
		Where("upload_id = ? AND NOT (? = ANY(completed))", uploadID, chunkIndex).
		Update("completed", gorm.Expr("array_append(completed, ?)", chunkIndex)).Error
}

func (r *ChunkRepository) UpdateStatus(uploadID string, status string) error {
	return r.db.Model(&model.ChunkUpload{}).
		Where("upload_id = ?", uploadID).
		Update("status", status).Error
}

func (r *ChunkRepository) Delete(uploadID string) error {
	return r.db.Where("upload_id = ?", uploadID).Delete(&model.ChunkUpload{}).Error
}

func (r *ChunkRepository) ListIncompleteByUser(userID uint64) ([]model.ChunkUpload, error) {
	var uploads []model.ChunkUpload
	err := r.db.Where("user_id = ? AND status = ?", userID, "uploading").
		Order("created_at DESC").
		Find(&uploads).Error
	return uploads, err
}

func (r *ChunkRepository) DeleteExpired(beforeTime interface{}) error {
	return r.db.Where("status IN ? AND created_at < ?", []string{"uploading", "cancelled"}, beforeTime).
		Delete(&model.ChunkUpload{}).Error
}

func (r *ChunkRepository) GetUploadID(id uint64) (*model.ChunkUpload, error) {
	var c model.ChunkUpload
	err := r.db.Where("id = ?", id).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpsertSystemConfig creates or updates a system config entry
func (r *ChunkRepository) GetSystemConfig(key string) (string, error) {
	var cfg model.SystemConfig
	err := r.db.Where("key = ?", key).First(&cfg).Error
	if err != nil {
		return "", err
	}
	return cfg.Value, nil
}

// ── system_config helpers (used by admin) ──

func (r *ChunkRepository) SetSystemConfig(key, value string) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&model.SystemConfig{Key: key, Value: value}).Error
}

func (r *ChunkRepository) GetAllSystemConfigs() ([]model.SystemConfig, error) {
	var cfgs []model.SystemConfig
	err := r.db.Find(&cfgs).Error
	return cfgs, err
}
