package model

import "time"

// Permission defines a single permission code.
type Permission struct {
	ID          uint64    `json:"id,string" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null;size:100"`
	Code        string    `json:"code" gorm:"uniqueIndex;not null;size:100"`
	Description string    `json:"description" gorm:"size:300"`
	GroupName   string    `json:"group_name" gorm:"size:50"`
	CreatedAt   time.Time `json:"created_at"`
}

// Role defines a named role grouping permissions.
type Role struct {
	ID          uint64       `json:"id,string" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"uniqueIndex;not null;size:50"`
	Code        string       `json:"code" gorm:"uniqueIndex;not null;size:50"`
	Description string       `json:"description" gorm:"size:200"`
	IsSystem    bool         `json:"is_system" gorm:"default:false"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time    `json:"created_at"`
}

// RolePermission is the join table between roles and permissions.
type RolePermission struct {
	RoleID       uint64 `json:"role_id,string" gorm:"primaryKey"`
	PermissionID uint64 `json:"permission_id,string" gorm:"primaryKey"`
}

// UserRole associates a user with a role.
type UserRole struct {
	UserID    uint64    `json:"user_id,string" gorm:"primaryKey"`
	RoleID    uint64    `json:"role_id,string" gorm:"primaryKey"`
	GrantedBy uint64    `json:"granted_by,string"`
	GrantedAt time.Time `json:"granted_at"`
}
