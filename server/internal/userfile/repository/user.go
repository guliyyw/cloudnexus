package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id uint64) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateUser(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) SaveRefreshToken(token *model.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *UserRepository) FindRefreshToken(token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.db.Where("token = ?", token).First(&rt).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *UserRepository) DeleteRefreshToken(token string) error {
	return r.db.Where("token = ?", token).Delete(&model.RefreshToken{}).Error
}

func (r *UserRepository) DeleteExpiredTokens() error {
	return r.db.Where("expires_at < now()", true).Delete(&model.RefreshToken{}).Error
}

func (r *UserRepository) FindUsers(page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	query := r.db.Model(&model.User{})
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("id ASC").Offset(offset).Limit(pageSize).Find(&users).Error
	return users, total, err
}

func (r *UserRepository) SetAdmin(id uint64, isAdmin bool) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("is_admin", isAdmin).Error
}

func (r *UserRepository) SetStatus(id uint64, status int8) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

func (r *UserRepository) UpdateFields(id uint64, fields map[string]interface{}) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Updates(fields).Error
}

func (r *UserRepository) SearchUsers(keyword string, page, pageSize int) ([]model.UserBrief, int64, error) {
	var users []model.User
	var total int64

	query := r.db.Model(&model.User{}).
		Where("deleted_at IS NULL AND status = 1 AND (privacy IS NULL OR privacy NOT LIKE '%\"allow_search\":false%')")

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username ILIKE ? OR email = ? OR nickname ILIKE ?", like, keyword, like)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("username ASC").Offset(offset).Limit(pageSize).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	results := make([]model.UserBrief, len(users))
	for i, u := range users {
		results[i] = model.UserBrief{
			ID:       u.ID,
			Username: u.Username,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Status:   u.Status,
		}
	}
	return results, total, nil
}

func (r *UserRepository) CountUsers() (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Count(&count).Error
	return count, err
}
