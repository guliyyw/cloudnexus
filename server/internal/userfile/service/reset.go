package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/cloudnexus/server/pkg/crypto"
	"github.com/cloudnexus/server/pkg/email"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type ResetService struct {
	db     *gorm.DB
	sender *email.Sender
}

func NewResetService(db *gorm.DB, sender *email.Sender) *ResetService {
	return &ResetService{db: db, sender: sender}
}

func (s *ResetService) RequestPasswordReset(emailAddr string) error {
	var user model.User
	if err := s.db.Where("email = ?", emailAddr).First(&user).Error; err != nil {
		// Don't reveal whether the email exists — silently succeed
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return apperrors.NewAppError(500, "生成令牌失败", err)
	}
	token := hex.EncodeToString(tokenBytes)

	rt := &model.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := s.db.Create(rt).Error; err != nil {
		return apperrors.NewAppError(500, "保存重置令牌失败", err)
	}

	subject := "CloudNexus 密码重置"
	body := "您正在重置 CloudNexus 账户密码。\n\n" +
		"请在浏览器中打开以下链接（30 分钟内有效）:\n" +
		"/reset-password?token=" + token + "\n\n" +
		"如果您没有请求重置密码，请忽略此邮件。"

	return s.sender.Send(emailAddr, subject, body)
}

func (s *ResetService) ResetPassword(token, newPassword string) error {
	if len(newPassword) < 8 {
		return apperrors.NewAppError(400, "密码至少 8 位", apperrors.ErrBadRequest)
	}

	var rt model.PasswordResetToken
	if err := s.db.Where("token = ? AND used = false AND expires_at > ?", token, time.Now()).First(&rt).Error; err != nil {
		return apperrors.NewAppError(400, "重置令牌无效或已过期", apperrors.ErrBadRequest)
	}

	hashed, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperrors.NewAppError(500, "密码加密失败", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", rt.UserID).Update("password", hashed).Error; err != nil {
			return err
		}
		if err := tx.Model(&rt).Update("used", true).Error; err != nil {
			return err
		}
		return nil
	})
}
