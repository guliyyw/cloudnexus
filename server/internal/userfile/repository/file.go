package repository

import (
	"errors"

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
	return r.db.Model(&model.File{}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).Update("deleted_at", gorm.Expr("now()")).Error
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

func (r *FileRepository) BatchSoftDelete(ids []uint64, userID uint64) (int64, error) {
	result := r.db.Model(&model.File{}).
		Where("id IN ? AND user_id = ? AND deleted_at IS NULL", ids, userID).
		Update("deleted_at", gorm.Expr("now()"))
	return result.RowsAffected, result.Error
}

func (r *FileRepository) Update(file *model.File) error {
	return r.db.Save(file).Error
}

func (r *FileRepository) FindByNameAndParent(userID, parentID uint64, name string) (*model.File, error) {
	var f model.File
	err := r.db.Where("user_id = ? AND parent_id = ? AND name = ? AND deleted_at IS NULL", userID, parentID, name).First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FileRepository) FindAllByParent(userID, parentID uint64) ([]model.File, error) {
	var files []model.File
	err := r.db.Where("user_id = ? AND parent_id = ? AND deleted_at IS NULL", userID, parentID).
		Order("is_dir DESC, name ASC").Find(&files).Error
	return files, err
}

func (r *FileRepository) BatchCreate(files []*model.File) error {
	return r.db.Create(&files).Error
}
