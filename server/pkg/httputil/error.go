// Package httputil provides HTTP-related utility functions for handlers.
package httputil

import (
	"net/http"

	"github.com/cloudnexus/server/pkg/logger"
	"github.com/cloudnexus/server/pkg/response"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HandleError handles errors in HTTP handlers and returns appropriate JSON response.
// It logs internal errors and returns safe error messages to clients.
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Check if it's an application error
	if appErr, ok := err.(*apperrors.AppError); ok {
		c.JSON(appErr.Code, response.Error(appErr.Code, appErr.Message))
		return
	}

	// Log internal errors
	logger.Log.Error("handler error",
		zap.Error(err),
		zap.String("path", c.Request.URL.Path),
		zap.Uint64("user_id", GetUserID(c)),
	)

	// Return generic error to client (don't expose internal details)
	c.JSON(http.StatusInternalServerError, response.Error(500, "服务器内部错误"))
}

// HandleErrorWithStatus handles errors with a specific HTTP status code.
func HandleErrorWithStatus(c *gin.Context, status int, err error) {
	if err == nil {
		return
	}

	logger.Log.Error("handler error",
		zap.Error(err),
		zap.String("path", c.Request.URL.Path),
		zap.Int("status", status),
		zap.Uint64("user_id", GetUserID(c)),
	)

	c.JSON(status, response.Error(status, err.Error()))
}

// BadRequest returns a 400 Bad Request response.
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, response.Error(400, message))
}

// Unauthorized returns a 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "未登录或登录已过期"
	}
	c.JSON(http.StatusUnauthorized, response.Error(401, message))
}

// Forbidden returns a 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "权限不足"
	}
	c.JSON(http.StatusForbidden, response.Error(403, message))
}

// NotFound returns a 404 Not Found response.
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "资源不存在"
	}
	c.JSON(http.StatusNotFound, response.Error(404, message))
}
