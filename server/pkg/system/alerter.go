package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cloudnexus/server/pkg/model"
	"gorm.io/gorm"
)

// AlertEvaluator evaluates alert rules and fires webhooks when nodes change status.
type AlertEvaluator struct {
	db         *gorm.DB
	cooldowns  map[string]time.Time
	cooldownsMu sync.Mutex
}

// NewAlertEvaluator creates a new AlertEvaluator.
func NewAlertEvaluator(db *gorm.DB) *AlertEvaluator {
	return &AlertEvaluator{
		db:        db,
		cooldowns: make(map[string]time.Time),
	}
}

type webhookPayload struct {
	RuleID    string `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	NodeName  string `json:"node_name"`
	NodeType  string `json:"node_type"`
	AlertType string `json:"alert_type"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	FiredAt   string `json:"fired_at"`
}

// Evaluate checks all enabled alert rules against the node and fires webhooks.
func (e *AlertEvaluator) Evaluate(node model.DockerNode, alertType string) {
	var rules []model.AlertRule
	if err := e.db.Where("enabled = ? AND trigger_type = ?", true, "status_change").Find(&rules).Error; err != nil {
		return
	}

	now := time.Now()
	for _, rule := range rules {
		if rule.NodeName != "*" && rule.NodeName != node.Name {
			continue
		}
		cooldownKey := fmt.Sprintf("%d:%s", rule.ID, node.Name)
		if e.inCooldown(cooldownKey, now, rule.CooldownSeconds) {
			continue
		}
		e.setCooldown(cooldownKey, now)

		msg := buildAlertMessage(node.Name, alertType)
		payload := webhookPayload{
			RuleID:    fmt.Sprintf("%d", rule.ID),
			RuleName:  rule.Name,
			NodeName:  node.Name,
			NodeType:  node.NodeType,
			AlertType: alertType,
			Status:    "firing",
			Message:   msg,
			FiredAt:   now.Format(time.RFC3339),
		}
		go e.fireWebhook(rule, payload)
	}
}

// ResolveAlert marks firing alerts for a node as resolved.
func (e *AlertEvaluator) ResolveAlert(nodeName string) {
	now := time.Now()
	e.db.Model(&model.AlertHistory{}).
		Where("node_name = ? AND status = 'firing' AND alert_type IN ('unresponsive','offline') AND resolved_at IS NULL", nodeName).
		Updates(map[string]interface{}{
			"status":      "resolved",
			"resolved_at": now,
		})
}

func buildAlertMessage(nodeName, alertType string) string {
	switch alertType {
	case "unresponsive":
		return fmt.Sprintf("节点 %s 无响应（连续2次探测失败）", nodeName)
	case "offline":
		return fmt.Sprintf("节点 %s 已离线（连续5次探测失败）", nodeName)
	case "recovery":
		return fmt.Sprintf("节点 %s 已恢复", nodeName)
	default:
		return fmt.Sprintf("节点 %s 状态变更: %s", nodeName, alertType)
	}
}

func (e *AlertEvaluator) inCooldown(key string, now time.Time, cooldownSec int) bool {
	e.cooldownsMu.Lock()
	defer e.cooldownsMu.Unlock()
	last, ok := e.cooldowns[key]
	if !ok {
		return false
	}
	return now.Sub(last) < time.Duration(cooldownSec)*time.Second
}

func (e *AlertEvaluator) setCooldown(key string, now time.Time) {
	e.cooldownsMu.Lock()
	defer e.cooldownsMu.Unlock()
	e.cooldowns[key] = now
}

func (e *AlertEvaluator) fireWebhook(rule model.AlertRule, payload webhookPayload) {
	body, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", rule.WebhookURL, bytes.NewReader(body))
	if err != nil {
		e.saveHistory(rule, payload, 0, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	statusCode := 0
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	} else {
		statusCode = resp.StatusCode
		resp.Body.Close()
	}
	e.saveHistory(rule, payload, statusCode, errMsg)
}

func (e *AlertEvaluator) saveHistory(rule model.AlertRule, payload webhookPayload, respCode int, errMsg string) {
	now := time.Now()
	history := model.AlertHistory{
		RuleID:       rule.ID,
		RuleName:     rule.Name,
		NodeName:     payload.NodeName,
		AlertType:    payload.AlertType,
		Status:       "firing",
		Message:      payload.Message,
		FiredAt:      now,
		WebhookURL:   rule.WebhookURL,
		ResponseCode: respCode,
		ErrorMessage: errMsg,
	}
	e.db.Create(&history)
}
