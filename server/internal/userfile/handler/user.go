package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/repository"
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
	quotaRepo  *repository.QuotaRepository
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) WithQuotaRepo(repo *repository.QuotaRepository) *UserHandler {
	h.quotaRepo = repo
	return h
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

	// SECURITY: 设置 httpOnly Cookie 存储 Token
	jwtCfg := h.svc.GetJWTConfig()
	auth.SetTokenCookies(c, pair, jwtCfg.AccessTTL, jwtCfg.RefreshTTL)

	c.JSON(http.StatusOK, response.OKWithData(pair))
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *UserHandler) HandleRefresh(c *gin.Context) {
	// SECURITY: 优先从 Cookie 获取 refresh_token
	refreshToken := auth.GetRefreshTokenFromCookie(c)
	if refreshToken == "" {
		// 兼容旧版：仍支持 JSON body 传递
		var req refreshReq
		if err := c.ShouldBindJSON(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, response.Error(400, "缺少刷新令牌"))
		return
	}
	pair, err := h.svc.RefreshToken(refreshToken)
	if err != nil {
		handleError(c, err)
		return
	}
	// SECURITY: 设置 httpOnly Cookie 存储 Token
	jwtCfg := h.svc.GetJWTConfig()
	auth.SetTokenCookies(c, pair, jwtCfg.AccessTTL, jwtCfg.RefreshTTL)
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

// HandleGetPermissions 返回当前用户的权限信息
// SECURITY: 前端通过此接口获取权限，而非客户端解析 JWT
func (h *UserHandler) HandleGetPermissions(c *gin.Context) {
	userID := c.GetUint64("user_id")
	username := c.GetString("username")
	isAdmin := c.GetBool("is_admin")
	roles, permissions := h.svc.GetUserRolesAndPermissions(userID)

	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"user_id":    userID,
		"username":   username,
		"is_admin":   isAdmin,
		"roles":      roles,
		"permissions": permissions,
	}))
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

// HandleLogout 处理用户登出，清除 httpOnly Cookie
func (h *UserHandler) HandleLogout(c *gin.Context) {
	// SECURITY: 清除 httpOnly Cookie
	auth.ClearTokenCookies(c)
	c.JSON(http.StatusOK, response.OK("已登出"))
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

// adminUserItem combines user info with quota info for admin list.
type adminUserItem struct {
	model.User
	StorageUsed  int64  `json:"storage_used"`
	StorageLimit *int64 `json:"storage_limit"`
	TierID       *uint64 `json:"tier_id,string"`
	TierName     string `json:"tier_name"`
}

func (h *UserHandler) HandleAdminListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	users, total, err := h.svc.ListUsers(page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]adminUserItem, 0, len(users))
	if len(users) > 0 {
		userIDs := make([]uint64, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}

		quotaMap := map[uint64]*model.UserQuota{}
		// SECURITY: 检查错误并记录日志
		if h.quotaRepo != nil {
			var err error
			quotaMap, err = h.quotaRepo.BatchFindUserQuotas(userIDs)
			if err != nil {
				// 使用标准 log 记录错误（handler 层未引入 zap）
				log.Printf("[WARN] 批量查询用户配额失败: %v", err)
			}
		}

		// 收集所有唯一的 TierID，避免 N+1 查询
		tierIDSet := make(map[uint64]struct{})
		for _, q := range quotaMap {
			if q.TierID != nil {
				tierIDSet[*q.TierID] = struct{}{}
			}
		}
		// 批量查询所有 Tier
		tierMap := make(map[uint64]*model.QuotaTier)
		if len(tierIDSet) > 0 && h.quotaRepo != nil {
			tierIDs := make([]uint64, 0, len(tierIDSet))
			for id := range tierIDSet {
				tierIDs = append(tierIDs, id)
			}
			tiers, err := h.quotaRepo.BatchFindTiersByIDs(tierIDs)
			if err != nil {
				log.Printf("[WARN] 批量查询配额等级失败: %v", err)
			} else {
				for i := range tiers {
					tierMap[tiers[i].ID] = &tiers[i]
				}
			}
		}

		for _, u := range users {
			item := adminUserItem{User: u}
			if q, ok := quotaMap[u.ID]; ok {
				item.StorageUsed = q.StorageUsed
				item.StorageLimit = q.StorageLimit
				item.TierID = q.TierID
				if q.TierID != nil {
					// 从内存 map 中获取，避免 N+1 查询
					if t, ok := tierMap[*q.TierID]; ok {
						item.TierName = t.Name
					}
				}
			}
			items = append(items, item)
		}
	}

	c.JSON(http.StatusOK, response.OKWithData(gin.H{
		"items":     items,
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
	// 防止管理员自我降权
	currentUserID := c.GetUint64("user_id")
	if userID == currentUserID {
		c.JSON(http.StatusForbidden, response.Error(403, "不能修改自己的管理员权限"))
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
