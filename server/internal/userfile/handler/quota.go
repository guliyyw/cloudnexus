package handler

import (
	"net/http"
	"strconv"

	"github.com/cloudnexus/server/internal/userfile/repository"
	"github.com/cloudnexus/server/internal/userfile/service"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
)

type QuotaHandler struct {
	svc      *service.QuotaService
	fileRepo *repository.FileRepository
}

func NewQuotaHandler(svc *service.QuotaService, fileRepo *repository.FileRepository) *QuotaHandler {
	return &QuotaHandler{svc: svc, fileRepo: fileRepo}
}

// HandleGetQuota returns the current user's quota info.
func (h *QuotaHandler) HandleGetQuota(c *gin.Context) {
	userID := c.GetUint64("user_id")
	info, err := h.svc.GetQuotaInfo(userID, h.fileRepo)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(info))
}

// ── Admin: Quota Tiers ──

func (h *QuotaHandler) HandleListTiers(c *gin.Context) {
	tiers, err := h.svc.ListTiers()
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(gin.H{"tiers": tiers}))
}

func (h *QuotaHandler) HandleCreateTier(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		StorageLimit  int64  `json:"storage_limit" binding:"required"`
		Description  string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误：需要 name 和 storage_limit"))
		return
	}
	if req.StorageLimit <= 0 {
		c.JSON(http.StatusBadRequest, response.Error(400, "storage_limit 必须大于 0"))
		return
	}
	tier, err := h.svc.CreateTier(req.Name, req.StorageLimit, req.Description)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.OKWithData(tier))
}

func (h *QuotaHandler) HandleUpdateTier(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的等级 ID"))
		return
	}
	var req struct {
		Name        *string `json:"name"`
		StorageLimit *int64  `json:"storage_limit"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.StorageLimit != nil {
		if *req.StorageLimit <= 0 {
			c.JSON(http.StatusBadRequest, response.Error(400, "storage_limit 必须大于 0"))
			return
		}
		updates["storage_limit"] = *req.StorageLimit
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, response.Error(400, "无更新内容"))
		return
	}
	if err := h.svc.UpdateTier(id, updates); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已更新"))
}

func (h *QuotaHandler) HandleDeleteTier(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的等级 ID"))
		return
	}
	if err := h.svc.DeleteTier(id); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("已删除"))
}

// HandleGetUserQuota returns quota info for a specific user (admin use).
func (h *QuotaHandler) HandleGetUserQuota(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	info, err := h.svc.GetUserQuotaDetail(userID, h.fileRepo)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OKWithData(info))
}

// HandleSetUserQuota allows admin to override a user's quota tier or limit.
func (h *QuotaHandler) HandleSetUserQuota(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "无效的用户 ID"))
		return
	}
	// Use raw map to detect which fields were actually sent
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}

	updates := make(map[string]interface{})
	if v, ok := raw["storage_limit"]; ok {
		if v == nil {
			updates["storage_limit"] = nil
		} else if f, ok := v.(float64); ok {
			updates["storage_limit"] = int64(f)
		}
	}
	if v, ok := raw["tier_id"]; ok {
		if v == nil {
			updates["tier_id"] = nil
		} else {
			switch val := v.(type) {
			case string:
				if id, err := strconv.ParseUint(val, 10, 64); err == nil {
					updates["tier_id"] = id
				}
			case float64:
				updates["tier_id"] = uint64(val)
			}
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, response.Error(400, "无更新内容"))
		return
	}
	if err := h.svc.SetUserQuota(userID, updates); err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.OK("配额已更新"))
}
