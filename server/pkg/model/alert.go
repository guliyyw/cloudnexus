package model

import "time"

// AlertRule defines when to fire alert webhooks.
type AlertRule struct {
	BaseModel
	Name            string `json:"name" gorm:"uniqueIndex;not null;size:128"`
	Description     string `json:"description" gorm:"size:512"`
	Enabled         bool   `json:"enabled" gorm:"default:true"`
	NodeName        string `json:"node_name" gorm:"default:'*';size:64"`
	TriggerType     string `json:"trigger_type" gorm:"default:'status_change';size:32"`
	Condition       string `json:"condition" gorm:"type:text"`
	WebhookURL      string `json:"webhook_url" gorm:"not null;size:512"`
	CooldownSeconds int    `json:"cooldown_seconds" gorm:"default:300"`
	CreatedBy       uint64 `json:"created_by,string" gorm:"default:0"`
}

func (AlertRule) TableName() string { return "alert_rules" }

// AlertHistory records past alert firings.
type AlertHistory struct {
	ID            uint64     `json:"id,string" gorm:"primaryKey"`
	RuleID        uint64     `json:"rule_id,string" gorm:"not null;index"`
	RuleName      string     `json:"rule_name" gorm:"not null;size:128"`
	NodeName      string     `json:"node_name" gorm:"not null;size:64;index"`
	AlertType     string     `json:"alert_type" gorm:"not null;size:32"`
	Status        string     `json:"status" gorm:"default:'firing';size:16"`
	Message       string     `json:"message" gorm:"type:text"`
	FiredAt       time.Time  `json:"fired_at" gorm:"not null;index"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	WebhookURL    string     `json:"webhook_url" gorm:"size:512"`
	ResponseCode  int        `json:"response_code" gorm:"default:0"`
	ErrorMessage  string     `json:"error_message" gorm:"type:text"`
}

func (AlertHistory) TableName() string { return "alert_history" }
