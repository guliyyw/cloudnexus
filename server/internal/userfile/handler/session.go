package handler

import (
	"net/http"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type SessionHandler struct {
	svc *service.SessionService
}

func NewSessionHandler(svc *service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

func (h *SessionHandler) HandleListSessions(c *gin.Context) {
	userID := c.GetUint64("user_id")
	sessions, err := h.svc.ListActiveSessions(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	if sessions == nil {
		sessions = []model.UserSession{}
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"sessions": sessions,
		"total":    len(sessions),
	}))
}

func (h *SessionHandler) HandleRevokeSession(c *gin.Context) {
	userID := c.GetUint64("user_id")
	jti := c.Param("jti")
	if jti == "" {
		c.JSON(http.StatusBadRequest, response.Error(400, "缺少会话标识"))
		return
	}

	currentJTI, _ := c.Get("jti")
	if currentJTIStr, ok := currentJTI.(string); ok && currentJTIStr == jti {
		c.JSON(http.StatusBadRequest, response.Error(400, "不能撤销当前会话，请使用全部下线功能"))
		return
	}

	// SECURITY: 增加用户归属验证，确保只能撤销自己的会话
	if err := h.svc.RevokeSessionByUser(jti, userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("会话已下线"))
}

func (h *SessionHandler) HandleRevokeAllSessions(c *gin.Context) {
	userID := c.GetUint64("user_id")
	currentJTI, _ := c.Get("jti")
	currentJTIStr, _ := currentJTI.(string)

	if err := h.svc.RevokeAllSessions(userID, currentJTIStr); err != nil {
		handleError(c, err)
		return
	}
	if err := h.svc.UpdateUserForceLogout(userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("所有其他设备已下线"))
}
