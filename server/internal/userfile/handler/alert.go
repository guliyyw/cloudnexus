package handler

import (
	"strconv"

	"github.com/cloudnexus/server/pkg/model"
	"github.com/cloudnexus/server/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AlertHandler struct {
	db *gorm.DB
}

func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{db: db}
}

type createRuleReq struct {
	Name            string `json:"name" binding:"required,min=1,max=128"`
	Description     string `json:"description"`
	Enabled         *bool  `json:"enabled"`
	NodeName        string `json:"node_name"`
	TriggerType     string `json:"trigger_type"`
	Condition       string `json:"condition"`
	WebhookURL      string `json:"webhook_url" binding:"required"`
	CooldownSeconds int    `json:"cooldown_seconds"`
}

type updateRuleReq struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	Enabled         *bool   `json:"enabled"`
	NodeName        *string `json:"node_name"`
	TriggerType     *string `json:"trigger_type"`
	Condition       *string `json:"condition"`
	WebhookURL      *string `json:"webhook_url"`
	CooldownSeconds *int    `json:"cooldown_seconds"`
}

// HandleListRules returns all alert rules.
func (h *AlertHandler) HandleListRules(c *gin.Context) {
	var rules []model.AlertRule
	if err := h.db.Order("created_at DESC").Find(&rules).Error; err != nil {
		c.JSON(500, response.Error(500, "查询告警规则失败"))
		return
	}
	if rules == nil {
		rules = []model.AlertRule{}
	}
	c.JSON(200, response.OKWithData(gin.H{"rules": rules}))
}

// HandleCreateRule creates a new alert rule.
func (h *AlertHandler) HandleCreateRule(c *gin.Context) {
	var req createRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误: name 和 webhook_url 必填"))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	nodeName := req.NodeName
	if nodeName == "" {
		nodeName = "*"
	}
	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = "status_change"
	}
	cooldown := req.CooldownSeconds
	if cooldown <= 0 {
		cooldown = 300
	}

	createdBy := uint64(0)
	if uid, ok := c.Get("user_id"); ok {
		if v, ok := uid.(uint64); ok {
			createdBy = v
		}
	}

	rule := model.AlertRule{
		Name:            req.Name,
		Description:     req.Description,
		Enabled:         enabled,
		NodeName:        nodeName,
		TriggerType:     triggerType,
		Condition:       req.Condition,
		WebhookURL:      req.WebhookURL,
		CooldownSeconds: cooldown,
		CreatedBy:       createdBy,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		c.JSON(409, response.Error(409, "规则名已存在或创建失败"))
		return
	}
	c.JSON(201, response.OKWithData(gin.H{"rule": rule}))
}

// HandleUpdateRule updates an existing alert rule.
func (h *AlertHandler) HandleUpdateRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的规则 ID"))
		return
	}

	var req updateRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, response.Error(400, "参数错误"))
		return
	}

	var rule model.AlertRule
	if err := h.db.First(&rule, id).Error; err != nil {
		c.JSON(404, response.Error(404, "规则不存在"))
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.NodeName != nil {
		updates["node_name"] = *req.NodeName
	}
	if req.TriggerType != nil {
		updates["trigger_type"] = *req.TriggerType
	}
	if req.Condition != nil {
		updates["condition"] = *req.Condition
	}
	if req.WebhookURL != nil {
		updates["webhook_url"] = *req.WebhookURL
	}
	if req.CooldownSeconds != nil {
		updates["cooldown_seconds"] = *req.CooldownSeconds
	}

	if len(updates) == 0 {
		c.JSON(200, response.OKWithData(gin.H{"rule": rule}))
		return
	}

	if err := h.db.Model(&rule).Updates(updates).Error; err != nil {
		c.JSON(409, response.Error(409, "更新失败，名称可能重复"))
		return
	}
	h.db.First(&rule, id)
	c.JSON(200, response.OKWithData(gin.H{"rule": rule}))
}

// HandleDeleteRule deletes an alert rule.
func (h *AlertHandler) HandleDeleteRule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(400, response.Error(400, "无效的规则 ID"))
		return
	}
	if err := h.db.Delete(&model.AlertRule{}, id).Error; err != nil {
		c.JSON(500, response.Error(500, "删除规则失败"))
		return
	}
	c.JSON(200, response.OK("规则已删除"))
}

// HandleListHistory returns alert history with pagination and filters.
func (h *AlertHandler) HandleListHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.db.Model(&model.AlertHistory{})
	if ruleID := c.Query("rule_id"); ruleID != "" {
		if id, err := strconv.ParseUint(ruleID, 10, 64); err == nil {
			query = query.Where("rule_id = ?", id)
		}
	}
	if nodeName := c.Query("node_name"); nodeName != "" {
		query = query.Where("node_name = ?", nodeName)
	}
	if alertType := c.Query("alert_type"); alertType != "" {
		query = query.Where("alert_type = ?", alertType)
	}

	var total int64
	query.Count(&total)

	var items []model.AlertHistory
	query.Order("fired_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items)

	if items == nil {
		items = []model.AlertHistory{}
	}

	c.JSON(200, response.OKWithData(gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}
