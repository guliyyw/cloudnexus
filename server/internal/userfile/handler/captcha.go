package handler

import (
	"net/http"

	"github.com/cloudnexus/server/pkg/captcha"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type CaptchaHandler struct {
	mgr *captcha.Manager
}

func NewCaptchaHandler(mgr *captcha.Manager) *CaptchaHandler {
	return &CaptchaHandler{mgr: mgr}
}

func (h *CaptchaHandler) HandleGenerate(c *gin.Context) {
	id, b64s, err := h.mgr.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "验证码生成失败"))
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"captcha_id":    id,
		"image_base64": b64s,
	}))
}

type verifyCaptchaReq struct {
	CaptchaID  string `json:"captcha_id" binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
}

func (h *CaptchaHandler) HandleVerify(c *gin.Context) {
	var req verifyCaptchaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	valid := h.mgr.Verify(req.CaptchaID, req.CaptchaCode)
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"valid": valid,
	}))
}
