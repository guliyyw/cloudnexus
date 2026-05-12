package service

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type BlocklistService struct {
	db *gorm.DB
}

func NewBlocklistService(db *gorm.DB) *BlocklistService {
	return &BlocklistService{db: db}
}

func (s *BlocklistService) BlockUser(userID, blockedUserID uint64, reason string) error {
	b := &model.Blocklist{
		UserID:        userID,
		BlockedUserID: blockedUserID,
		Reason:        reason,
	}
	return s.db.Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		FirstOrCreate(b).Error
}

func (s *BlocklistService) UnblockUser(userID, blockedUserID uint64) error {
	return s.db.Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).
		Delete(&model.Blocklist{}).Error
}

func (s *BlocklistService) IsBlocked(userID, targetID uint64) bool {
	var count int64
	s.db.Model(&model.Blocklist{}).
		Where("(user_id = ? AND blocked_user_id = ?) OR (user_id = ? AND blocked_user_id = ?)",
			userID, targetID, targetID, userID).
		Count(&count)
	return count > 0
}

func (s *BlocklistService) GetBlocklist(userID uint64) ([]model.Blocklist, error) {
	var list []model.Blocklist
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&list).Error
	return list, err
}
