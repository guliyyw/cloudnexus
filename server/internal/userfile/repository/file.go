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

func (r *FileRepository) FindByIDIncludingDeleted(id uint64) (*model.File, error) {
	var f model.File
	err := r.db.Unscoped().Where("id = ?", id).First(&f).Error
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

func (r *FileRepository) ForceDelete(id uint64) error {
	return r.db.Unscoped().Delete(&model.File{}, id).Error
}

// ── 文件版本 ──

func (r *FileRepository) CreateVersion(v *model.FileVersion) error {
	return r.db.Create(v).Error
}

func (r *FileRepository) DeleteVersion(id uint64) error {
	return r.db.Delete(&model.FileVersion{}, id).Error
}

func (r *FileRepository) ListVersions(fileID uint64, page, pageSize int) ([]model.FileVersion, int64, error) {
	var versions []model.FileVersion
	var total int64

	query := r.db.Model(&model.FileVersion{}).Where("file_id = ?", fileID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("version_num DESC").Offset(offset).Limit(pageSize).Find(&versions).Error
	return versions, total, err
}

func (r *FileRepository) GetMaxVersionNum(fileID uint64) (int, error) {
	var maxNum int
	err := r.db.Model(&model.FileVersion{}).
		Select("COALESCE(MAX(version_num), 0)").
		Where("file_id = ?", fileID).
		Scan(&maxNum).Error
	return maxNum, err
}

func (r *FileRepository) FindVersionByID(id uint64) (*model.FileVersion, error) {
	var v model.FileVersion
	err := r.db.Where("id = ?", id).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ── 回收站 ──

func (r *FileRepository) FindDeletedByUser(userID uint64, page, pageSize int) ([]model.File, int64, error) {
	var files []model.File
	var total int64
	query := r.db.Model(&model.File{}).Where("user_id = ? AND deleted_at IS NOT NULL", userID)
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("deleted_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error
	return files, total, err
}

func (r *FileRepository) RestoreFromTrash(id, userID uint64) error {
	return r.db.Model(&model.File{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NOT NULL", id, userID).
		Update("deleted_at", nil).Error
}

func (r *FileRepository) FindDeletedExpired(before interface{}, limit int) ([]model.File, error) {
	var files []model.File
	err := r.db.Where("deleted_at IS NOT NULL AND deleted_at < ?", before).
		Limit(limit).Find(&files).Error
	return files, err
}

func (r *FileRepository) BatchForceDelete(ids []uint64) error {
	return r.db.Unscoped().Where("id IN ?", ids).Delete(&model.File{}).Error
}

// ── 空间统计 ──

func (r *FileRepository) SumSizeByUser(userID uint64) (int64, error) {
	var size int64
	err := r.db.Model(&model.File{}).
		Where("user_id = ? AND deleted_at IS NULL AND is_dir = false", userID).
		Select("COALESCE(SUM(size), 0)").
		Scan(&size).Error
	return size, err
}

func (r *FileRepository) SumDeletedSizeByUser(userID uint64) (int64, error) {
	var size int64
	err := r.db.Model(&model.File{}).
		Where("user_id = ? AND deleted_at IS NOT NULL AND is_dir = false", userID).
		Select("COALESCE(SUM(size), 0)").
		Scan(&size).Error
	return size, err
}

func (r *FileRepository) SumVersionSizeByUser(userID uint64) (int64, error) {
	var size int64
	err := r.db.Model(&model.FileVersion{}).
		Joins("JOIN files ON files.id = file_versions.file_id").
		Where("files.user_id = ? AND files.deleted_at IS NULL", userID).
		Select("COALESCE(SUM(file_versions.size), 0)").
		Scan(&size).Error
	return size, err
}

func (r *FileRepository) ForceDeleteVersionsByFileID(fileID uint64) error {
	return r.db.Where("file_id = ?", fileID).Delete(&model.FileVersion{}).Error
}

func (r *FileRepository) CleanupFileVersionsByFileID(fileID uint64) error {
	return r.db.Where("file_id = ?", fileID).Delete(&model.FileVersion{}).Error
}
