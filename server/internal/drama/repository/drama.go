package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DramaRepository struct {
	db *gorm.DB
}

func NewDramaRepository(db *gorm.DB) *DramaRepository {
	return &DramaRepository{db: db}
}

func (r *DramaRepository) ListProjects(ownerID uint64, keyword, sort string, page, pageSize int) ([]model.DramaProject, int64, error) {
	var projects []model.DramaProject
	var total int64
	query := r.db.Model(&model.DramaProject{}).Where("owner_id = ?", ownerID)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR description LIKE ?", like, like)
	}
	query.Count(&total)

	order := "updated_at DESC"
	switch strings.ToLower(sort) {
	case "created_asc":
		order = "created_at ASC"
	case "created_desc":
		order = "created_at DESC"
	case "title_asc":
		order = "title ASC"
	case "title_desc":
		order = "title DESC"
	}
	offset := (page - 1) * pageSize
	err := query.Order(order).Offset(offset).Limit(pageSize).Find(&projects).Error
	return projects, total, err
}

func (r *DramaRepository) CreateProject(project *model.DramaProject) error {
	return r.db.Create(project).Error
}

func (r *DramaRepository) GetProject(ownerID, id uint64) (*model.DramaProject, error) {
	var project model.DramaProject
	if err := r.db.Where("id = ? AND owner_id = ?", id, ownerID).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *DramaRepository) UpdateProject(project *model.DramaProject) error {
	return r.db.Save(project).Error
}

func (r *DramaRepository) DeleteProject(ownerID, id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("owner_id = ? AND project_id = ?", ownerID, id).Delete(&model.DramaTask{}).Error; err != nil {
			return err
		}
		if err := tx.Where("owner_id = ? AND project_id = ?", ownerID, id).Delete(&model.DramaAsset{}).Error; err != nil {
			return err
		}
		if err := tx.Where("owner_id = ? AND project_id = ?", ownerID, id).Delete(&model.DramaStoryboard{}).Error; err != nil {
			return err
		}
		return tx.Where("owner_id = ? AND id = ?", ownerID, id).Delete(&model.DramaProject{}).Error
	})
}

func (r *DramaRepository) ReplaceStoryboards(ownerID, projectID uint64, storyboards []model.DramaStoryboard) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("owner_id = ? AND project_id = ?", ownerID, projectID).Delete(&model.DramaStoryboard{}).Error; err != nil {
			return err
		}
		if len(storyboards) == 0 {
			return nil
		}
		return tx.Create(&storyboards).Error
	})
}

func (r *DramaRepository) ListStoryboards(ownerID, projectID uint64) ([]model.DramaStoryboard, error) {
	var storyboards []model.DramaStoryboard
	err := r.db.Where("owner_id = ? AND project_id = ?", ownerID, projectID).Order("seq ASC").Find(&storyboards).Error
	return storyboards, err
}

func (r *DramaRepository) GetStoryboard(ownerID, projectID, id uint64) (*model.DramaStoryboard, error) {
	var storyboard model.DramaStoryboard
	err := r.db.Where("owner_id = ? AND project_id = ? AND id = ?", ownerID, projectID, id).First(&storyboard).Error
	if err != nil {
		return nil, err
	}
	return &storyboard, nil
}

func (r *DramaRepository) UpdateStoryboard(storyboard *model.DramaStoryboard) error {
	return r.db.Save(storyboard).Error
}

func (r *DramaRepository) UpsertAssets(assets []model.DramaAsset) error {
	if len(assets) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "type"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"description", "updated_at"}),
	}).Create(&assets).Error
}

func (r *DramaRepository) ReplaceAssets(ownerID, projectID uint64, assets []model.DramaAsset) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("owner_id = ? AND project_id = ?", ownerID, projectID).Delete(&model.DramaAsset{}).Error; err != nil {
			return err
		}
		if len(assets) == 0 {
			return nil
		}
		return tx.Create(&assets).Error
	})
}

func (r *DramaRepository) ListAssets(ownerID, projectID uint64) ([]model.DramaAsset, error) {
	var assets []model.DramaAsset
	err := r.db.Where("owner_id = ? AND project_id = ?", ownerID, projectID).Order("type ASC, name ASC").Find(&assets).Error
	return assets, err
}

func (r *DramaRepository) GetAsset(ownerID, projectID, id uint64) (*model.DramaAsset, error) {
	var asset model.DramaAsset
	err := r.db.Where("owner_id = ? AND project_id = ? AND id = ?", ownerID, projectID, id).First(&asset).Error
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *DramaRepository) UpdateAsset(asset *model.DramaAsset) error {
	return r.db.Save(asset).Error
}

func (r *DramaRepository) CreateFile(file *model.File) error {
	return r.db.Create(file).Error
}

func (r *DramaRepository) FindFileByNameAndParent(ownerID, parentID uint64, name string) (*model.File, error) {
	var file model.File
	err := r.db.Where("user_id = ? AND parent_id = ? AND name = ? AND deleted_at IS NULL", ownerID, parentID, name).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *DramaRepository) ListFilesByParent(ownerID, parentID uint64) ([]model.File, error) {
	var files []model.File
	err := r.db.Where("user_id = ? AND parent_id = ? AND deleted_at IS NULL", ownerID, parentID).Find(&files).Error
	return files, err
}

func (r *DramaRepository) ListFilesByStoragePrefix(ownerID uint64, prefix string) ([]model.File, error) {
	var files []model.File
	err := r.db.Where("user_id = ? AND storage_key LIKE ? AND deleted_at IS NULL", ownerID, prefix+"%").Find(&files).Error
	return files, err
}

func (r *DramaRepository) SoftDeleteFiles(ownerID uint64, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return r.db.Model(&model.File{}).Where("user_id = ? AND id IN ? AND deleted_at IS NULL", ownerID, ids).Update("deleted_at", &now).Error
}

func (r *DramaRepository) ListTasks(ownerID, projectID uint64) ([]model.DramaTask, error) {
	var tasks []model.DramaTask
	err := r.db.Where("owner_id = ? AND project_id = ?", ownerID, projectID).Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (r *DramaRepository) CreateTask(task *model.DramaTask) error {
	return r.db.Create(task).Error
}

func (r *DramaRepository) GetSetting(ownerID uint64) (*model.DramaSetting, error) {
	var setting model.DramaSetting
	err := r.db.Where("owner_id = ?", ownerID).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = model.DramaSetting{
			OwnerID:       ownerID,
			TTSEngine:     "edge-tts",
			TTSConfig:     "{}",
			VideoSettings: `{"resolution":"1080p","fps":30,"bitrate":"8M","subtitle":{"font":"Microsoft YaHei","size":42,"color":"#FFFFFF","outline":"#000000","position":"bottom"}}`,
			StorageRoot:   "短剧工坊",
		}
		if createErr := r.db.Create(&setting).Error; createErr != nil {
			return nil, createErr
		}
		return &setting, nil
	}
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *DramaRepository) SaveSetting(setting *model.DramaSetting) error {
	return r.db.Save(setting).Error
}
