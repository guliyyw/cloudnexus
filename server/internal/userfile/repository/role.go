package repository

import (
	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

type RoleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) FindRolesByUserID(userID uint64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Table("roles").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

func (r *RoleRepository) FindPermissionsByUserID(userID uint64) ([]model.Permission, error) {
	var perms []model.Permission
	err := r.db.Table("permissions").
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Joins("JOIN user_roles ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ?", userID).
		Distinct().
		Find(&perms).Error
	return perms, err
}

func (r *RoleRepository) ListRoles() ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Preload("Permissions").Find(&roles).Error
	return roles, err
}

func (r *RoleRepository) FindRoleByID(id uint64) (*model.Role, error) {
	var role model.Role
	err := r.db.Preload("Permissions").First(&role, id).Error
	return &role, err
}

func (r *RoleRepository) CreateRole(role *model.Role) error {
	return r.db.Create(role).Error
}

func (r *RoleRepository) UpdateRole(role *model.Role) error {
	return r.db.Save(role).Error
}

func (r *RoleRepository) DeleteRole(id uint64) error {
	return r.db.Delete(&model.Role{}, id).Error
}

func (r *RoleRepository) ListPermissions() ([]model.Permission, error) {
	var perms []model.Permission
	err := r.db.Find(&perms).Error
	return perms, err
}

func (r *RoleRepository) AssignRole(userID, roleID, grantedBy uint64) error {
	return r.db.Create(&model.UserRole{
		UserID:    userID,
		RoleID:    roleID,
		GrantedBy: grantedBy,
	}).Error
}

func (r *RoleRepository) RemoveRole(userID, roleID uint64) error {
	return r.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserRole{}).Error
}

func (r *RoleRepository) FindUserRoles(userID uint64) ([]model.Role, error) {
	return r.FindRolesByUserID(userID)
}

func (r *RoleRepository) CountRoles() (int64, error) {
	var count int64
	err := r.db.Model(&model.Role{}).Count(&count).Error
	return count, err
}

func (r *RoleRepository) BatchCreatePermissions(perms []model.Permission) error {
	if len(perms) == 0 {
		return nil
	}
	return r.db.Create(&perms).Error
}

func (r *RoleRepository) BatchCreateRoles(roles []model.Role) error {
	if len(roles) == 0 {
		return nil
	}
	return r.db.Create(&roles).Error
}

func (r *RoleRepository) FindRoleByCode(code string) (*model.Role, error) {
	var role model.Role
	err := r.db.Where("code = ?", code).First(&role).Error
	return &role, err
}

func (r *RoleRepository) AssignRolePermissions(roleID uint64, permIDs []uint64) error {
	// Delete existing assignments and re-create
	if err := r.db.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
		return err
	}
	for _, pid := range permIDs {
		if err := r.db.Create(&model.RolePermission{
			RoleID:       roleID,
			PermissionID: pid,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
