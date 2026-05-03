package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type FileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(file *model.File) error {
	return r.db.Create(file).Error
}

func (r *FileRepository) FindByID(id uint64) (*model.File, error) {
	var f model.File
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FileRepository) FindByUserAndParent(userID, parentID uint64, page, pageSize int) ([]model.File, int64, error) {
	var files []model.File
	var total int64

	query := r.db.Model(&model.File{}).Where("user_id = ? AND parent_id = ? AND deleted_at IS NULL", userID, parentID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("is_dir DESC, name ASC").Offset(offset).Limit(pageSize).Find(&files).Error
	return files, total, err
}

func (r *FileRepository) SoftDelete(id, userID uint64) error {
	return r.db.Model(&model.File{}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).Update("deleted_at", gorm.Expr("now()", true)).Error
}

func (r *FileRepository) SearchFiles(userID uint64, keyword string, page, pageSize int) ([]model.File, int64, error) {
	var files []model.File
	var total int64

	like := "%" + keyword + "%"
	query := r.db.Model(&model.File{}).Where("user_id = ? AND deleted_at IS NULL AND name LIKE ?", userID, like)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("is_dir DESC, name ASC").Offset(offset).Limit(pageSize).Find(&files).Error
	return files, total, err
}
