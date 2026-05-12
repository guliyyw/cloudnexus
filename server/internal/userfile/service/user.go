package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"github.com/cloudnexus/server/internal/userfile/repository"
	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/crypto"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"unicode"
)

type UserService struct {
	repo      *repository.UserRepository
	roleRepo  *repository.RoleRepository
	jwtConfig auth.Config
}

func NewUserService(repo *repository.UserRepository, jwtConfig auth.Config) *UserService {
	return &UserService{repo: repo, jwtConfig: jwtConfig}
}

func (s *UserService) WithRoleRepo(r *repository.RoleRepository) *UserService {
	s.roleRepo = r
	return s
}

func (s *UserService) getUserRolesAndPermissions(userID uint64) ([]string, []string) {
	if s.roleRepo == nil {
		return []string{}, []string{}
	}
	roles, err := s.roleRepo.FindRolesByUserID(userID)
	if err != nil {
		return []string{}, []string{}
	}
	roleCodes := make([]string, len(roles))
	for i, r := range roles {
		roleCodes[i] = r.Code
	}
	perms, err := s.roleRepo.FindPermissionsByUserID(userID)
	if err != nil {
		return roleCodes, []string{}
	}
	permCodes := make([]string, len(perms))
	for i, p := range perms {
		permCodes[i] = p.Code
	}
	return roleCodes, permCodes
}

func (s *UserService) GetJWTConfig() auth.Config {
	return s.jwtConfig
}

func (s *UserService) Register(username, email, password string) (*model.User, error) {
	if _, err := s.repo.FindByUsername(username); err == nil {
		return nil, apperrors.NewAppError(409, "用户名已存在", apperrors.ErrConflict)
	}
	if _, err := s.repo.FindByEmail(email); err == nil {
		return nil, apperrors.NewAppError(409, "邮箱已被注册", apperrors.ErrConflict)
	}

	hashed, err := crypto.HashPassword(password)
	if err != nil {
		return nil, apperrors.NewAppError(500, "密码加密失败", err)
	}

	user := &model.User{
		Username: username,
		Email:    email,
		Password: hashed,
	}
	if err := s.repo.CreateUser(user); err != nil {
		return nil, apperrors.NewAppError(500, "创建用户失败", err)
	}
	return user, nil
}

func (s *UserService) Login(username, password string) (*auth.TokenPair, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, apperrors.NewAppError(401, "用户名或密码错误", apperrors.ErrUnauthorized)
	}

	// Check lockout
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return nil, apperrors.NewAppError(429, "账户已被锁定，请稍后再试", apperrors.ErrUnauthorized)
	}

	if !crypto.CheckPassword(password, user.Password) {
		// Track failure count
		failCount := user.LoginFailCount + 1
		updates := map[string]interface{}{"login_fail_count": failCount}
		if failCount >= 5 {
			locked := time.Now().Add(15 * time.Minute)
			updates["locked_until"] = locked
		}
		s.repo.UpdateFields(user.ID, updates)
		return nil, apperrors.NewAppError(401, "用户名或密码错误", apperrors.ErrUnauthorized)
	}

	// Reset failure count on success
	s.repo.UpdateFields(user.ID, map[string]interface{}{
		"login_fail_count": 0,
		"locked_until":     nil,
	})

	roles, perms := s.getUserRolesAndPermissions(user.ID)
	pair, err := auth.GenerateTokenPair(s.jwtConfig, user.ID, user.Username, user.IsAdmin, roles, perms)
	if err != nil {
		return nil, apperrors.NewAppError(500, "令牌生成失败", err)
	}

	refreshToken := &model.RefreshToken{
		UserID:    user.ID,
		Token:     hashToken(pair.RefreshToken),
		ExpiresAt: time.Now().Add(s.jwtConfig.RefreshTTL),
	}
	if err := s.repo.SaveRefreshToken(refreshToken); err != nil {
		return nil, apperrors.NewAppError(500, "保存令牌失败", err)
	}

	return pair, nil
}

func (s *UserService) RefreshToken(rawToken string) (*auth.TokenPair, error) {
	hashed := hashToken(rawToken)
	rt, err := s.repo.FindRefreshToken(hashed)
	if err != nil {
		return nil, apperrors.NewAppError(401, "刷新令牌无效", apperrors.ErrUnauthorized)
	}
	if time.Now().After(rt.ExpiresAt) {
		s.repo.DeleteRefreshToken(hashed)
		return nil, apperrors.NewAppError(401, "刷新令牌已过期", apperrors.ErrUnauthorized)
	}
	s.repo.DeleteRefreshToken(hashed)

	claims, err := auth.ParseToken(rawToken, s.jwtConfig.RefreshSecret)
	if err != nil {
		return nil, apperrors.NewAppError(401, "刷新令牌解析失败", apperrors.ErrUnauthorized)
	}

	roles, perms := s.getUserRolesAndPermissions(claims.UserID)
	pair, err := auth.GenerateTokenPair(s.jwtConfig, claims.UserID, claims.Username, claims.IsAdmin, roles, perms)
	if err != nil {
		return nil, apperrors.NewAppError(500, "令牌生成失败", err)
	}

	newRT := &model.RefreshToken{
		UserID:    claims.UserID,
		Token:     hashToken(pair.RefreshToken),
		ExpiresAt: time.Now().Add(s.jwtConfig.RefreshTTL),
	}
	if err := s.repo.SaveRefreshToken(newRT); err != nil {
		return nil, apperrors.NewAppError(500, "保存令牌失败", err)
	}

	return pair, nil
}

func (s *UserService) GetProfile(userID uint64) (*model.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)
	}
	return user, nil
}

func (s *UserService) UpdateProfile(userID uint64, email, avatar string) (*model.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)
	}
	if email != "" {
		user.Email = email
	}
	if avatar != "" {
		user.Avatar = avatar
	}
	if err := s.repo.UpdateUser(user); err != nil {
		return nil, apperrors.NewAppError(500, "更新用户失败", err)
	}
	return user, nil
}

func (s *UserService) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)
	}
	if !crypto.CheckPassword(oldPassword, user.Password) {
		return apperrors.NewAppError(400, "原密码错误", apperrors.ErrBadRequest)
	}
	hashed, err := crypto.HashPassword(newPassword)
	if err != nil {
		return apperrors.NewAppError(500, "密码加密失败", err)
	}
	user.Password = hashed
	return s.repo.UpdateUser(user)
}

func (s *UserService) ListUsers(page, pageSize int) ([]model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	return s.repo.FindUsers(page, pageSize)
}

func (s *UserService) ToggleAdmin(userID uint64) (*model.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)
	}
	user.IsAdmin = !user.IsAdmin
	if err := s.repo.SetAdmin(userID, user.IsAdmin); err != nil {
		return nil, apperrors.NewAppError(500, "更新失败", err)
	}
	return user, nil
}

func (s *UserService) ToggleStatus(userID uint64) (*model.User, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)
	}
	newStatus := int8(0)
	if user.Status == 0 {
		newStatus = 1
	}
	user.Status = newStatus
	if err := s.repo.SetStatus(userID, newStatus); err != nil {
		return nil, apperrors.NewAppError(500, "更新失败", err)
	}
	return user, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *UserService) SeedDefaultAdmin() {
	count, err := s.repo.CountUsers()
	if err != nil {
		return
	}
	if count > 0 {
		return
	}

	username := os.Getenv("DEFAULT_ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}
	email := os.Getenv("DEFAULT_ADMIN_EMAIL")
	if email == "" {
		email = "admin@cloudnexus.local"
	}
	password := os.Getenv("DEFAULT_ADMIN_PASSWORD")
	if password == "" {
		password = "CloudNexus@admin"
	}

	hashed, err := crypto.HashPassword(password)
	if err != nil {
		return
	}

	user := &model.User{
		Username: username,
		Email:    email,
		Password: hashed,
		IsAdmin:  true,
	}
	if err := s.repo.CreateUser(user); err != nil {
		return
	}
}

func (s *UserService) GetPrivacy(userID uint64) (*model.UserPrivacy, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, apperrors.NewAppError(404, "用户不存在", apperrors.ErrNotFound)
	}
	privacy := &model.UserPrivacy{AllowSearch: true, AllowAddFriend: true, ShowOnline: true}
	if user.Privacy != "" {
		json.Unmarshal([]byte(user.Privacy), privacy)
	}
	return privacy, nil
}

func (s *UserService) UpdatePrivacy(userID uint64, privacy *model.UserPrivacy) error {
	data, err := json.Marshal(privacy)
	if err != nil {
		return apperrors.NewAppError(400, "隐私设置格式错误", apperrors.ErrBadRequest)
	}
	return s.repo.UpdateFields(userID, map[string]interface{}{"privacy": string(data)})
}

// ValidatePasswordStrength checks that the password meets minimum requirements:
// 8+ chars, at least one uppercase, one lowercase, one digit, one special char.
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return apperrors.NewAppError(400, "密码至少 8 位", apperrors.ErrBadRequest)
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return apperrors.NewAppError(400, "密码需包含大写字母、小写字母、数字和特殊字符", apperrors.ErrBadRequest)
	}
	return nil
}
