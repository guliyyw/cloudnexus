package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudnexus/server/pkg/config"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type SessionService struct {
	db  *gorm.DB
	rdb *redis.Client
	cfg config.AppConfig
}

func NewSessionService(db *gorm.DB, rdb *redis.Client, cfg config.AppConfig) *SessionService {
	return &SessionService{db: db, rdb: rdb, cfg: cfg}
}

func (s *SessionService) CreateSession(userID uint64, jti, userAgent, ipAddr string) error {
	expiresAt := time.Now().Add(time.Duration(s.cfg.JWT.AccessTTL) * time.Second)
	sess := &model.UserSession{
		UserID:       userID,
		JTI:          jti,
		UserAgent:    userAgent,
		IPAddress:    ipAddr,
		LoginAt:      time.Now(),
		LastActiveAt: time.Now(),
		ExpiresAt:    expiresAt,
	}
	return s.db.Create(sess).Error
}

func (s *SessionService) ValidateSession(jti string) (bool, error) {
	if s.rdb == nil {
		// No Redis — only check DB
		return s.checkDBSession(jti)
	}

	// Check Redis blacklist
	blacklisted, err := s.rdb.Get(context.Background(), fmt.Sprintf("jti:%s", jti)).Result()
	if err == nil && blacklisted == "revoked" {
		return false, nil
	}

	return s.checkDBSession(jti)
}

func (s *SessionService) checkDBSession(jti string) (bool, error) {
	var sess model.UserSession
	err := s.db.Where("jti = ? AND is_active = true AND expires_at > ?", jti, time.Now()).First(&sess).Error
	if err != nil {
		return false, nil
	}
	// Update last_active_at
	s.db.Model(&sess).Update("last_active_at", time.Now())
	return true, nil
}

func (s *SessionService) RevokeSession(jti string) error {
	if err := s.db.Model(&model.UserSession{}).Where("jti = ?", jti).Update("is_active", false).Error; err != nil {
		return err
	}
	if s.rdb != nil {
		// Add to Redis blacklist with remaining TTL
		var sess model.UserSession
		if err := s.db.Where("jti = ?", jti).First(&sess).Error; err == nil {
			ttl := time.Until(sess.ExpiresAt)
			if ttl > 0 {
				s.rdb.Set(context.Background(), fmt.Sprintf("jti:%s", jti), "revoked", ttl)
			}
		}
	}
	return nil
}

func (s *SessionService) RevokeAllSessions(userID uint64, exceptJTI string) error {
	result := s.db.Model(&model.UserSession{}).
		Where("user_id = ? AND is_active = true AND jti != ?", userID, exceptJTI).
		Update("is_active", false)
	return result.Error
}

func (s *SessionService) ListActiveSessions(userID uint64) ([]model.UserSession, error) {
	var sessions []model.UserSession
	err := s.db.Where("user_id = ? AND is_active = true AND expires_at > ?", userID, time.Now()).
		Order("login_at DESC").Find(&sessions).Error
	return sessions, err
}

func (s *SessionService) CleanExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			s.db.Where("expires_at < ? OR (is_active = false AND last_active_at < ?)",
				time.Now(), time.Now().Add(-7*24*time.Hour)).
				Delete(&model.UserSession{})
		}
	}()
}

func (s *SessionService) UpdateUserForceLogout(userID uint64) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).
		Update("force_logout_after", time.Now()).Error
}

func (s *SessionService) IsForceLogout(userID uint64, loginAt time.Time) (bool, error) {
	var user model.User
	if err := s.db.Select("force_logout_after").Where("id = ?", userID).First(&user).Error; err != nil {
		return false, apperrors.NewAppError(404, "用户不存在", err)
	}
	if user.ForceLogoutAfter != nil && user.ForceLogoutAfter.After(loginAt) {
		return true, nil
	}
	return false, nil
}
