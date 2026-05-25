package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TokenRevoker is used to check whether a JWT (identified by JTI) has been revoked.
// Services that have Redis can pass a Redis-backed implementation.
type TokenRevoker interface {
	IsRevoked(ctx context.Context, jti string) bool
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = c.GetHeader("X-Request-ID")
		}
		if requestID == "" {
			b := make([]byte, 6)
			rand.Read(b)
			requestID = hex.EncodeToString(b)
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-Id", requestID)

		c.Next()

		latency := time.Since(start)
		userID := c.GetUint64("user_id")
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", c.Writer.Size()),
		}
		if requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}
		if userID != 0 {
			fields = append(fields, zap.Uint64("user_id", userID))
		}

		log := logger.Log
		if c.Writer.Status() >= 500 {
			log.Error("request completed", fields...)
		} else if c.Writer.Status() >= 400 {
			log.Warn("request completed", fields...)
		} else {
			log.Info("request completed", fields...)
		}
	}
}

func CORS() gin.HandlerFunc {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost"
	}
	allowedSet := make(map[string]bool)
	for _, o := range strings.Split(allowedOrigins, ",") {
		allowedSet[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" || allowedSet[origin] {
			if origin == "" {
				origin = "*"
			}
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func AuthRequired(secret string, revoker ...TokenRevoker) gin.HandlerFunc {
	var tokenRevoker TokenRevoker
	if len(revoker) > 0 {
		tokenRevoker = revoker[0]
	}
	return func(c *gin.Context) {
		// SECURITY: 优先从 Authorization Header 获取 token
		header := c.GetHeader("Authorization")
		token := header
		if len(token) > 7 && strings.EqualFold(token[:7], "Bearer ") {
			token = token[7:]
		}
		// 其次从查询参数获取（兼容 WebSocket 等场景）
		if token == "" {
			token = c.Query("token")
		}
		// 最后从 httpOnly Cookie 获取
		if token == "" {
			token, _ = c.Cookie("access_token")
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "缺少认证令牌"})
			return
		}
		claims, err := auth.ParseToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "令牌无效或已过期"})
			return
		}

		// 检查 JTI 是否已被吊销（如强制下线）
		if tokenRevoker != nil && claims.JTI != "" {
			if tokenRevoker.IsRevoked(c.Request.Context(), claims.JTI) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "令牌已被吊销"})
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("jti", claims.JTI)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Next()
	}
}

// CheckWebSocketOrigin validates WebSocket upgrade requests against ALLOWED_ORIGINS.
// Defaults to localhost origins if env var is not set.
func CheckWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // non-browser clients
	}
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost"
	}
	for _, o := range strings.Split(allowedOrigins, ",") {
		if strings.TrimSpace(o) == origin {
			return true
		}
	}
	return false
}

// AdminChecker 定义检查用户管理员状态的接口
type AdminChecker interface {
	IsAdmin(userID uint64) (bool, error)
}

// AdminRequired 基础版：仅检查 JWT 中的 is_admin 字段
// 注意：此方法不验证数据库中的实时权限状态
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin := c.GetBool("is_admin")
		if !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "需要管理员权限"})
			return
		}
		c.Next()
	}
}

// AdminRequiredWithDB 增强版：实时验证数据库中的管理员状态
// 需要传入 AdminChecker 实现（通常是 UserRepository）
func AdminRequiredWithDB(checker AdminChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint64("user_id")
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未登录"})
			return
		}

		// 实时验证数据库中的管理员状态
		isAdmin, err := checker.IsAdmin(userID)
		if err != nil {
			logger.Log.Error("验证管理员状态失败", zap.Error(err), zap.Uint64("user_id", userID))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "权限验证失败"})
			return
		}

		if !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "需要管理员权限"})
			return
		}
		c.Next()
	}
}

// SimpleDBAdminChecker 是基于 gorm.DB 的简单 AdminChecker 实现
type SimpleDBAdminChecker struct {
	db *gorm.DB
}

func NewSimpleDBAdminChecker(db *gorm.DB) *SimpleDBAdminChecker {
	return &SimpleDBAdminChecker{db: db}
}

func (c *SimpleDBAdminChecker) IsAdmin(userID uint64) (bool, error) {
	var isAdmin bool
	err := c.db.Model(&struct{ IsAdmin bool }{IsAdmin: false}).
		Table("users").
		Select("is_admin").
		Where("id = ?", userID).
		Scan(&isAdmin).Error
	return isAdmin, err
}
