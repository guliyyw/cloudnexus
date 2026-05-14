package service

import (
	"fmt"

	"github.com/cloudnexus/server/internal/userfile/repository"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
)

type QuotaService struct {
	repo *repository.QuotaRepository
}

func NewQuotaService(repo *repository.QuotaRepository) *QuotaService {
	return &QuotaService{repo: repo}
}

// QuotaInfo is the frontend-facing quota view.
type QuotaInfo struct {
	Used         int64   `json:"used"`
	Limit        int64   `json:"limit"`
	TierName     string  `json:"tier_name"`
	TrashUsed    int64   `json:"trash_used"`
	TrashLimit   int64   `json:"trash_limit"`
	UsagePercent float64 `json:"usage_percent"`
}

// CheckQuota verifies that adding additionalSize bytes won't exceed the user's limit.
func (s *QuotaService) CheckQuota(userID uint64, additionalSize int64) error {
	quota, err := s.repo.GetOrCreateQuota(userID)
	if err != nil {
		return apperrors.NewAppError(500, "读取配额信息失败", err)
	}

	tiers, err := s.repo.GetTierLimitMap()
	if err != nil {
		return apperrors.NewAppError(500, "读取配额等级失败", err)
	}

	limit := quota.GetEffectiveLimit(tiers)
	if quota.StorageUsed+additionalSize > limit {
		usedMB := quota.StorageUsed / 1024 / 1024
		limitMB := limit / 1024 / 1024
		return apperrors.NewAppError(413, fmt.Sprintf("存储空间不足，已用 %dMB/%dMB", usedMB, limitMB), apperrors.ErrBadRequest)
	}
	return nil
}

// CheckTrashSpace verifies that adding additionalSize bytes to trash won't exceed limit.
func (s *QuotaService) CheckTrashSpace(userID uint64, additionalSize int64, fileRepo *repository.FileRepository) error {
	trashSize, _ := fileRepo.SumDeletedSizeByUser(userID)
	const trashLimit = 1 * 1024 * 1024 * 1024 // 1GB
	if trashSize+additionalSize > trashLimit {
		return apperrors.NewAppError(413, "回收站已满，请先清理后再删除", apperrors.ErrBadRequest)
	}
	return nil
}

// UpdateUsed adjusts the user's storage_used by delta.
func (s *QuotaService) UpdateUsed(userID uint64, delta int64) error {
	return s.repo.AddStorageUsed(userID, delta)
}

// GetQuotaInfo returns the user's current quota overview.
func (s *QuotaService) GetQuotaInfo(userID uint64, fileRepo *repository.FileRepository) (*QuotaInfo, error) {
	quota, err := s.repo.GetOrCreateQuota(userID)
	if err != nil {
		return nil, err
	}

	tiers, err := s.repo.GetTierLimitMap()
	if err != nil {
		return nil, err
	}

	limit := quota.GetEffectiveLimit(tiers)

	tierName := "free"
	if quota.TierID != nil {
		if t, err := s.repo.FindTierByID(*quota.TierID); err == nil {
			tierName = t.Name
		}
	}

	trashUsed, _ := fileRepo.SumDeletedSizeByUser(userID)
	const trashLimit = 1 * 1024 * 1024 * 1024

	usagePercent := float64(0)
	if limit > 0 {
		usagePercent = float64(quota.StorageUsed) / float64(limit) * 100
	}

	return &QuotaInfo{
		Used:         quota.StorageUsed,
		Limit:        limit,
		TierName:     tierName,
		TrashUsed:    trashUsed,
		TrashLimit:   trashLimit,
		UsagePercent: usagePercent,
	}, nil
}

// ReconcileAll recalculates storage_used for all users with quotas.
func (s *QuotaService) ReconcileAll() error {
	ids, err := s.repo.ListAllUserIDsWithQuota()
	if err != nil {
		return err
	}
	for _, userID := range ids {
		used, err := s.repo.RecalculateStorageUsed(userID)
		if err != nil {
			continue
		}
		_ = s.repo.UpdateStorageUsed(userID, used)
	}
	return nil
}

// ── Tier management ──

func (s *QuotaService) ListTiers() ([]model.QuotaTier, error) {
	return s.repo.ListTiers()
}

func (s *QuotaService) CreateTier(name string, storageLimit int64, description string) (*model.QuotaTier, error) {
	tier := &model.QuotaTier{
		Name:         name,
		StorageLimit: storageLimit,
		Description:  description,
	}
	if err := s.repo.CreateTier(tier); err != nil {
		return nil, apperrors.NewAppError(500, "创建配额等级失败", err)
	}
	return tier, nil
}

func (s *QuotaService) UpdateTier(id uint64, updates map[string]interface{}) error {
	return s.repo.UpdateTier(id, updates)
}

func (s *QuotaService) DeleteTier(id uint64) error {
	count, err := s.repo.CountUsersWithTier(id)
	if err != nil {
		return apperrors.NewAppError(500, "查询等级使用情况失败", err)
	}
	if count > 0 {
		return apperrors.NewAppError(400, fmt.Sprintf("该等级被 %d 个用户使用，无法删除", count), apperrors.ErrConflict)
	}
	return s.repo.DeleteTier(id)
}

// SetUserQuota overrides a user's quota tier and/or custom limit.
func (s *QuotaService) SetUserQuota(userID uint64, storageLimit *int64, tierID *uint64) error {
	_, err := s.repo.GetOrCreateQuota(userID)
	if err != nil {
		return err
	}
	return s.repo.UpdateUserQuota(userID, storageLimit, tierID)
}
