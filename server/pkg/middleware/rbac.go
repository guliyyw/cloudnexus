package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
