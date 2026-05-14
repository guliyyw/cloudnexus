package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type QuotaRepository struct {
	db *gorm.DB
}

func NewQuotaRepository(db *gorm.DB) *QuotaRepository {
	return &QuotaRepository{db: db}
}

// ── User Quota ──

func (r *QuotaRepository) FindUserQuota(userID uint64) (*model.UserQuota, error) {
	var q model.UserQuota
	err := r.db.Where("user_id = ?", userID).First(&q).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QuotaRepository) CreateUserQuota(quota *model.UserQuota) error {
	return r.db.Create(quota).Error
}

func (r *QuotaRepository) UpdateStorageUsed(userID uint64, used int64) error {
	return r.db.Model(&model.UserQuota{}).
		Where("user_id = ?", userID).
		Update("storage_used", used).Error
}

func (r *QuotaRepository) AddStorageUsed(userID uint64, delta int64) error {
	return r.db.Model(&model.UserQuota{}).
		Where("user_id = ?", userID).
		Update("storage_used", gorm.Expr("storage_used + ?", delta)).Error
}

func (r *QuotaRepository) GetOrCreateQuota(userID uint64) (*model.UserQuota, error) {
	q, err := r.FindUserQuota(userID)
	if err == nil {
		return q, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	quota := &model.UserQuota{
		UserID:  userID,
		TierID:  uint64Ptr(1),
	}
	if err := r.CreateUserQuota(quota); err != nil {
		return nil, err
	}
	return quota, nil
}

func (r *QuotaRepository) RecalculateStorageUsed(userID uint64) (int64, error) {
	var fileSize int64
	r.db.Model(&model.File{}).
		Where("user_id = ? AND deleted_at IS NULL AND is_dir = false", userID).
		Select("COALESCE(SUM(size), 0)").
		Scan(&fileSize)

	var versionSize int64
	r.db.Model(&model.FileVersion{}).
		Joins("JOIN files ON files.id = file_versions.file_id").
		Where("files.user_id = ? AND files.deleted_at IS NULL", userID).
		Select("COALESCE(SUM(file_versions.size), 0)").
		Scan(&versionSize)

	return fileSize + versionSize, nil
}

func (r *QuotaRepository) ListAllUserIDsWithQuota() ([]uint64, error) {
	var ids []uint64
	err := r.db.Model(&model.UserQuota{}).Pluck("user_id", &ids).Error
	return ids, err
}

func (r *QuotaRepository) UpdateUserQuota(userID uint64, storageLimit *int64, tierID *uint64) error {
	updates := map[string]interface{}{
		"storage_limit": storageLimit,
		"tier_id":       tierID,
	}
	return r.db.Model(&model.UserQuota{}).Where("user_id = ?", userID).Updates(updates).Error
}

// ── Quota Tiers ──

func (r *QuotaRepository) ListTiers() ([]model.QuotaTier, error) {
	var tiers []model.QuotaTier
	err := r.db.Order("storage_limit ASC").Find(&tiers).Error
	return tiers, err
}

func (r *QuotaRepository) FindTierByID(id uint64) (*model.QuotaTier, error) {
	var t model.QuotaTier
	err := r.db.Where("id = ?", id).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *QuotaRepository) CreateTier(tier *model.QuotaTier) error {
	return r.db.Create(tier).Error
}

func (r *QuotaRepository) UpdateTier(id uint64, updates map[string]interface{}) error {
	return r.db.Model(&model.QuotaTier{}).Where("id = ?", id).Updates(updates).Error
}

func (r *QuotaRepository) DeleteTier(id uint64) error {
	return r.db.Delete(&model.QuotaTier{}, id).Error
}

func (r *QuotaRepository) CountUsersWithTier(tierID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&model.UserQuota{}).Where("tier_id = ?", tierID).Count(&count).Error
	return count, err
}

func (r *QuotaRepository) GetTierLimitMap() (map[uint64]int64, error) {
	tiers, err := r.ListTiers()
	if err != nil {
		return nil, err
	}
	m := make(map[uint64]int64, len(tiers))
	for _, t := range tiers {
		m[t.ID] = t.StorageLimit
	}
	return m, nil
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}
