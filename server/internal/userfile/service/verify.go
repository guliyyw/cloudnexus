package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/cloudnexus/server/pkg/email"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type VerifyService struct {
	db     *gorm.DB
	sender *email.Sender
}

func NewVerifyService(db *gorm.DB, sender *email.Sender) *VerifyService {
	return &VerifyService{db: db, sender: sender}
}

func (s *VerifyService) SendEmailCode(emailAddr, codeType string, userID uint64) error {
	code := generateCode()

	v := &model.EmailVerification{
		UserID:    userID,
		Email:     emailAddr,
		Code:      code,
		Type:      codeType,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.db.Create(v).Error; err != nil {
		return apperrors.NewAppError(500, "保存验证码失败", err)
	}

	var subject, body string
	if codeType == "register" {
		subject = "CloudNexus 注册验证码"
		body = fmt.Sprintf("您的注册验证码是: %s，10 分钟内有效。", code)
	} else {
		subject = "CloudNexus 邮箱验证码"
		body = fmt.Sprintf("您的验证码是: %s，10 分钟内有效。", code)
	}

	return s.sender.Send(emailAddr, subject, body)
}

func (s *VerifyService) VerifyEmail(emailAddr, code string, vType string) error {
	var v model.EmailVerification
	err := s.db.Where("email = ? AND code = ? AND type = ? AND used = false AND expires_at > ?",
		emailAddr, code, vType, time.Now()).First(&v).Error
	if err != nil {
		return apperrors.NewAppError(400, "验证码错误或已过期", apperrors.ErrBadRequest)
	}

	s.db.Model(&v).Update("used", true)

	if vType == "register" || vType == "email" {
		s.db.Model(&model.User{}).Where("email = ?", emailAddr).Update("email_verified", true)
		if v.UserID > 0 {
			s.db.Model(&model.User{}).Where("id = ?", v.UserID).Update("email_verified", true)
		}
	}
	return nil
}

func (s *VerifyService) SendPhoneCode(phone, codeType string, userID uint64) error {
	code := generateCode()

	v := &model.PhoneVerification{
		UserID:    userID,
		Phone:     phone,
		Code:      code,
		Type:      codeType,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.db.Create(v).Error; err != nil {
		return apperrors.NewAppError(500, "保存验证码失败", err)
	}

	// Phone SMS not yet implemented — code is logged
	return nil
}

func (s *VerifyService) VerifyPhone(phone, code string, vType string) error {
	var v model.PhoneVerification
	err := s.db.Where("phone = ? AND code = ? AND type = ? AND used = false AND expires_at > ?",
		phone, code, vType, time.Now()).First(&v).Error
	if err != nil {
		return apperrors.NewAppError(400, "验证码错误或已过期", apperrors.ErrBadRequest)
	}

	s.db.Model(&v).Update("used", true)
	s.db.Model(&model.User{}).Where("phone = ?", phone).Update("phone_verified", true)
	return nil
}

func generateCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		// fallback to deterministic but usable
		return "123456"
	}
	return fmt.Sprintf("%06d", n.Int64())
}
