package middleware

import (
	"net/http"

	"github.com/cloudnexus/server/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LoadPermissions refreshes roles and permissions from the database for the
// authenticated request, so grants and revocations take effect immediately.
func LoadPermissions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "Permission service unavailable"})
			return
		}
		userID := c.GetUint64("user_id")
		if userID == 0 {
			c.Next()
			return
		}
		var isAdmin bool
		if err := db.Table("users").Select("is_admin").Where("id = ?", userID).Scan(&isAdmin).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to load user access"})
			return
		}
		var roles []string
		if err := db.Table("roles").Select("roles.code").
			Joins("JOIN user_roles ur ON ur.role_id = roles.id").Where("ur.user_id = ?", userID).
			Pluck("roles.code", &roles).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to load roles"})
			return
		}
		var permissions []string
		if err := db.Raw(`
			SELECT DISTINCT p.code FROM permissions p
			LEFT JOIN role_permissions rp ON rp.permission_id = p.id
			LEFT JOIN user_roles ur ON ur.role_id = rp.role_id AND ur.user_id = ?
			LEFT JOIN user_permissions up ON up.permission_id = p.id AND up.user_id = ?
			WHERE ur.user_id IS NOT NULL OR up.user_id IS NOT NULL
		`, userID, userID).Scan(&permissions).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to load permissions"})
			return
		}
		c.Set("is_admin", isAdmin)
		c.Set("roles", roles)
		c.Set("permissions", permissions)
		c.Next()
	}
}

// PermissionChecker 定义权限检查接口
type PermissionChecker interface {
	// HasPermission 检查用户是否有指定权限
	HasPermission(userID uint64, perm string) (bool, error)
	// HasRole 检查用户是否有指定角色
	HasRole(userID uint64, role string) (bool, error)
	// IsAdmin 检查用户是否为管理员
	IsAdmin(userID uint64) (bool, error)
}

// HasPermission checks if the gin context contains the given permission.
func HasPermission(c *gin.Context, perm string) bool {
	perms := c.GetStringSlice("permissions")
	for _, p := range perms {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}

// HasRole checks if the gin context contains the given role.
func HasRole(c *gin.Context, role string) bool {
	roles := c.GetStringSlice("roles")
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// RequirePermission returns middleware that requires a specific permission.
// 注意：此方法仅检查 JWT 中的权限，不验证数据库实时状态
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("is_admin") {
			c.Next()
			return
		}
		if !HasPermission(c, perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "权限不足: " + perm,
			})
			return
		}
		c.Next()
	}
}

// RequirePermissionWithDB returns middleware that requires a specific permission with DB validation.
// 实时验证数据库中的权限状态
func RequirePermissionWithDB(checker PermissionChecker, perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint64("user_id")
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
			return
		}

		// 先检查是否为管理员
		isAdmin, err := checker.IsAdmin(userID)
		if err != nil {
			logger.Log.Error("验证管理员状态失败", zap.Error(err), zap.Uint64("user_id", userID))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "权限验证失败"})
			return
		}
		if isAdmin {
			c.Next()
			return
		}

		// 检查具体权限
		hasPerm, err := checker.HasPermission(userID, perm)
		if err != nil {
			logger.Log.Error("验证权限失败", zap.Error(err), zap.Uint64("user_id", userID), zap.String("perm", perm))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "权限验证失败"})
			return
		}

		if !hasPerm {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "权限不足: " + perm,
			})
			return
		}
		c.Next()
	}
}

// RequireAnyPermission returns middleware that requires at least one of the given permissions.
func RequireAnyPermission(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("is_admin") {
			c.Next()
			return
		}
		for _, p := range perms {
			if HasPermission(c, p) {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "权限不足",
		})
	}
}

// RequireRole returns middleware that requires a specific role.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("is_admin") {
			c.Next()
			return
		}
		if !HasRole(c, role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "需要角色: " + role,
			})
			return
		}
		c.Next()
	}
}

// RequireRoleWithDB returns middleware that requires a specific role with DB validation.
func RequireRoleWithDB(checker PermissionChecker, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint64("user_id")
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
			return
		}

		// 先检查是否为管理员
		isAdmin, err := checker.IsAdmin(userID)
		if err != nil {
			logger.Log.Error("验证管理员状态失败", zap.Error(err), zap.Uint64("user_id", userID))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "权限验证失败"})
			return
		}
		if isAdmin {
			c.Next()
			return
		}

		// 检查角色
		hasRole, err := checker.HasRole(userID, role)
		if err != nil {
			logger.Log.Error("验证角色失败", zap.Error(err), zap.Uint64("user_id", userID), zap.String("role", role))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "权限验证失败"})
			return
		}

		if !hasRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "需要角色: " + role,
			})
			return
		}
		c.Next()
	}
}
