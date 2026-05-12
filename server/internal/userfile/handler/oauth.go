package handler

import (
	"net/http"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type OAuthHandler struct {
	svc *service.OAuthService
}

func NewOAuthHandler(svc *service.OAuthService) *OAuthHandler {
	return &OAuthHandler{svc: svc}
}

func (h *OAuthHandler) HandleListBindings(c *gin.Context) {
	userID := c.GetUint64("user_id")
	bindings, err := h.svc.ListBindings(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(bindings))
}

type unbindOAuthReq struct {
	Provider string `json:"provider" binding:"required"`
}

func (h *OAuthHandler) HandleUnbind(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req unbindOAuthReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	if err := h.svc.UnbindOAuth(userID, req.Provider); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("解绑成功"))
}
