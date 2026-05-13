package handler

import (
	"net/http"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type VerifyHandler struct {
	svc *service.VerifyService
}

func NewVerifyHandler(svc *service.VerifyService) *VerifyHandler {
	return &VerifyHandler{svc: svc}
}

type sendEmailCodeReq struct {
	Email string `json:"email" binding:"required,email"`
}

func (h *VerifyHandler) HandleSendEmailCode(c *gin.Context) {
	var req sendEmailCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	userID := uint64(0)
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint64)
	}
	if err := h.svc.SendEmailCode(req.Email, "email", userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("验证码已发送"))
}

type verifyEmailReq struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

func (h *VerifyHandler) HandleVerifyEmail(c *gin.Context) {
	var req verifyEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	if err := h.svc.VerifyEmail(req.Email, req.Code, "email"); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("邮箱验证成功"))
}

type sendPhoneCodeReq struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *VerifyHandler) HandleSendPhoneCode(c *gin.Context) {
	var req sendPhoneCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	userID := uint64(0)
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint64)
	}
	if err := h.svc.SendPhoneCode(req.Phone, "phone", userID); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("验证码已发送"))
}

type verifyPhoneReq struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required,len=6"`
}

func (h *VerifyHandler) HandleVerifyPhone(c *gin.Context) {
	var req verifyPhoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	if err := h.svc.VerifyPhone(req.Phone, req.Code, "phone"); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("手机验证成功"))
}
