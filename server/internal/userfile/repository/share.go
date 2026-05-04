package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type ShareRepository struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) *ShareRepository {
	return &ShareRepository{db: db}
}

func (r *ShareRepository) Create(share *model.FileShare) error {
	return r.db.Create(share).Error
}

func (r *ShareRepository) FindByCode(code string) (*model.FileShare, error) {
	var share model.FileShare
	err := r.db.Where("share_code = ?", code).First(&share).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
}

func (r *ShareRepository) FindByID(id uint64) (*model.FileShare, error) {
	var share model.FileShare
	err := r.db.First(&share, id).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
}

func (r *ShareRepository) FindByFileID(fileID uint64) ([]model.FileShare, error) {
	var shares []model.FileShare
	err := r.db.Where("file_id = ?", fileID).Order("created_at DESC").Find(&shares).Error
	return shares, err
}

func (r *ShareRepository) FindByOwnerID(ownerID uint64) ([]model.FileShare, error) {
	var shares []model.FileShare
	err := r.db.Where("owner_id = ?", ownerID).Order("created_at DESC").Find(&shares).Error
	return shares, err
}

func (r *ShareRepository) Delete(id uint64) error {
	return r.db.Delete(&model.FileShare{}, id).Error
}

func (r *ShareRepository) IncrementDownloadCount(id uint64) error {
	return r.db.Model(&model.FileShare{}).Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + 1")).Error
}
