// Package httputil provides HTTP-related utility functions for handlers.
package httputil

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetUserID extracts the user ID from the gin context.
// Returns 0 if not found or invalid.
func GetUserID(c *gin.Context) uint64 {
	uid, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	if id, ok := uid.(uint64); ok {
		return id
	}
	return 0
}

// GetUsername extracts the username from the gin context.
// Returns empty string if not found.
func GetUsername(c *gin.Context) string {
	uname, exists := c.Get("username")
	if !exists {
		return ""
	}
	if name, ok := uname.(string); ok {
		return name
	}
	return ""
}

// IsAdmin checks if the current user is an admin from the gin context.
func IsAdmin(c *gin.Context) bool {
	isAdmin, exists := c.Get("is_admin")
	if !exists {
		return false
	}
	if admin, ok := isAdmin.(bool); ok {
		return admin
	}
	return false
}

// GetUserIDString extracts the user ID from the gin context as string.
// Returns empty string if not found.
func GetUserIDString(c *gin.Context) string {
	uid := GetUserID(c)
	if uid == 0 {
		return ""
	}
	// Convert uint64 to string
	return strconv.FormatUint(uid, 10)
}

// MustGetUserID extracts the user ID and panics if not found.
// Use only when user ID is guaranteed to exist (e.g., after AuthRequired middleware).
func MustGetUserID(c *gin.Context) uint64 {
	uid := GetUserID(c)
	if uid == 0 {
		panic("user_id not found in context")
	}
	return uid
}
