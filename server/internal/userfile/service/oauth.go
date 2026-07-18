package service

import (
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type tokenCipher interface {
	Encrypt(plain string) (string, error)
	Decrypt(encrypted string) (string, error)
}

type OAuthService struct {
	db     *gorm.DB
	cipher tokenCipher
}

func NewOAuthService(db *gorm.DB, cipher tokenCipher) *OAuthService {
	return &OAuthService{db: db, cipher: cipher}
}

func (s *OAuthService) BindOAuth(userID uint64, provider, openID, accessToken, refreshToken string) error {
	encryptedAccessToken, err := s.cipher.Encrypt(accessToken)
	if err != nil {
		return apperrors.NewAppError(500, "加密 OAuth access token 失败", apperrors.ErrInternalServer)
	}
	encryptedRefreshToken, err := s.cipher.Encrypt(refreshToken)
	if err != nil {
		return apperrors.NewAppError(500, "加密 OAuth refresh token 失败", apperrors.ErrInternalServer)
	}
	binding := &model.OAuthBinding{
		UserID:                 userID,
		Provider:               provider,
		OpenID:                 openID,
		AccessToken:            encryptedAccessToken,
		RefreshToken:           encryptedRefreshToken,
		TokenEncryptionVersion: 1,
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
	if err := s.decryptBinding(&binding); err != nil {
		return nil, err
	}
	return &binding, nil
}

func (s *OAuthService) ListBindings(userID uint64) ([]model.OAuthBinding, error) {
	var bindings []model.OAuthBinding
	err := s.db.Where("user_id = ?", userID).Find(&bindings).Error
	return bindings, err
}

func (s *OAuthService) decryptBinding(binding *model.OAuthBinding) error {
	if binding.TokenEncryptionVersion == 0 {
		return nil
	}
	accessToken, err := s.cipher.Decrypt(binding.AccessToken)
	if err != nil {
		return apperrors.NewAppError(500, "解密 OAuth access token 失败", apperrors.ErrInternalServer)
	}
	refreshToken, err := s.cipher.Decrypt(binding.RefreshToken)
	if err != nil {
		return apperrors.NewAppError(500, "解密 OAuth refresh token 失败", apperrors.ErrInternalServer)
	}
	binding.AccessToken = accessToken
	binding.RefreshToken = refreshToken
	return nil
}
