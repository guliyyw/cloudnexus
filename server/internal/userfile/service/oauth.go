package service

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

// TODO: OAuth 令牌应加密存储
// 当前 AccessToken/RefreshToken 明文存储在数据库中，存在泄露风险
// 建议实现：
// 1. 在配置中添加 oauth_encryption_key
// 2. 存储 AES-GCM 加密后的令牌
// 3. 读取时解密

type OAuthService struct {
	db *gorm.DB
}

func NewOAuthService(db *gorm.DB) *OAuthService {
	return &OAuthService{db: db}
}

func (s *OAuthService) BindOAuth(userID uint64, provider, openID, accessToken, refreshToken string) error {
	// TODO: 加密 accessToken 和 refreshToken
	binding := &model.OAuthBinding{
		UserID:       userID,
		Provider:     provider,
		OpenID:       openID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return s.db.Where(model.OAuthBinding{Provider: provider, OpenID: openID}).
		Assign(binding).FirstOrCreate(binding).Error
}

func (s *OAuthService) UnbindOAuth(userID uint64, provider string) error {
	return s.db.Where("user_id = ? AND provider = ?", userID, provider).
		Delete(&model.OAuthBinding{}).Error
}

func (s *OAuthService) FindByOAuth(provider, openID string) (*model.OAuthBinding, error) {
	var binding model.OAuthBinding
	err := s.db.Where("provider = ? AND open_id = ?", provider, openID).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func (s *OAuthService) ListBindings(userID uint64) ([]model.OAuthBinding, error) {
	var bindings []model.OAuthBinding
	err := s.db.Where("user_id = ?", userID).Find(&bindings).Error
	return bindings, err
}
