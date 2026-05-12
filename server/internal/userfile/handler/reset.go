package handler

import (
	"net/http"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type ResetHandler struct {
	svc *service.ResetService
}

func NewResetHandler(svc *service.ResetService) *ResetHandler {
	return &ResetHandler{svc: svc}
}

type forgotPasswordReq struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *ResetHandler) HandleForgotPassword(c *gin.Context) {
	var req forgotPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	if err := h.svc.RequestPasswordReset(req.Email); err != nil {
		handleError(c, err)
		return
	}
	// Always return success to prevent email enumeration
	c.JSON(http.StatusOK, response.OK("如果该邮箱已注册，重置邮件已发送"))
}

type resetPasswordReq struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

func (h *ResetHandler) HandleResetPassword(c *gin.Context) {
	var req resetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	if err := h.svc.ResetPassword(req.Token, req.NewPassword); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("密码已重置，请使用新密码登录"))
}
