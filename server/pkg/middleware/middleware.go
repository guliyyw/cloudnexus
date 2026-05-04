package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-Id")
		if requestID == "" {
			requestID = c.GetHeader("X-Request-ID")
		}
		c.Header("X-Request-Id", requestID)

		c.Next()

		latency := time.Since(start)
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
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := header
		if len(token) > 7 && strings.EqualFold(token[:7], "Bearer ") {
			token = token[7:]
		} else if token == "" {
			token = c.Query("token")
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
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
