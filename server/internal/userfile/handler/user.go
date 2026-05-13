package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/auth"
	"github.com/cloudnexus/server/pkg/captcha"
	apperrors "github.com/cloudnexus/server/pkg/errors"
	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc        *service.UserService
	sessionSvc *service.SessionService
	captchaMgr *captcha.Manager
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) WithSessionService(svc *service.SessionService) *UserHandler {
	h.sessionSvc = svc
	return h
}

func (h *UserHandler) WithCaptchaManager(mgr *captcha.Manager) *UserHandler {
	h.captchaMgr = mgr
	return h
}

type registerReq struct {
	Username    string `json:"username" binding:"required,min=3,max=64"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8,max=128"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

func (h *UserHandler) HandleRegister(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}

	if h.captchaMgr != nil {
		if req.CaptchaID == "" || req.CaptchaCode == "" {
			c.JSON(http.StatusBadRequest, response.Error(400, "请完成验证码"))
			return
		}
		if !h.captchaMgr.Verify(req.CaptchaID, req.CaptchaCode) {
			c.JSON(http.StatusBadRequest, response.Error(400, "验证码错误或已过期"))
			return
		}
	}

	if err := service.ValidatePasswordStrength(req.Password); err != nil {
		handleError(c, err)
		return
	}

	user, err := h.svc.Register(req.Username, req.Email, req.Password)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(user))
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) HandleLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	pair, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		handleError(c, err)
		return
	}

	// Create session
	if h.sessionSvc != nil {
		claims, _ := auth.ParseToken(pair.AccessToken, h.svc.GetJWTConfig().AccessSecret)
		if claims != nil {
			h.sessionSvc.CreateSession(claims.UserID, claims.JTI,
				c.GetHeader("User-Agent"), c.ClientIP())
		}
	}

	c.JSON(http.StatusOK, response.OKWithData(pair))
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *UserHandler) HandleRefresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	pair, err := h.svc.RefreshToken(req.RefreshToken)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(pair))
}

func (h *UserHandler) HandleGetProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	user, err := h.svc.GetProfile(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(user))
}

type updateProfileReq struct {
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
}

func (h *UserHandler) HandleUpdateProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	user, err := h.svc.UpdateProfile(userID, req.Email, req.Avatar)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(user))
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
}

func (h *UserHandler) HandleChangePassword(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}
	if err := h.svc.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("密码已修改"))
}

func (h *UserHandler) HandleAdminListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	users, total, err := h.svc.ListUsers(page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

func (h *UserHandler) HandleAdminToggleAdmin(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	user, err := h.svc.ToggleAdmin(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(user))
}

func (h *UserHandler) HandleAdminToggleStatus(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	user, err := h.svc.ToggleStatus(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(user))
}

func (h *UserHandler) HandleGetPrivacy(c *gin.Context) {
	userID := c.GetUint64("user_id")
	privacy, err := h.svc.GetPrivacy(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(privacy))
}

func (h *UserHandler) HandleUpdatePrivacy(c *gin.Context) {
	userID := c.GetUint64("user_id")
	var privacy model.UserPrivacy
	if err := json.NewDecoder(c.Request.Body).Decode(&privacy); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := h.svc.UpdatePrivacy(userID, &privacy); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("隐私设置已更新"))
}

func handleError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		c.JSON(appErr.Code, response.Error(appErr.Code, appErr.Message))
		return
	}
	c.JSON(http.StatusInternalServerError, response.Error(500, "服务器内部错误"))
}
