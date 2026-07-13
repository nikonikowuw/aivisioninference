package dto

// AlertEventListRequest 告警事件列表查询参数
type AlertEventListRequest struct {
	PageRequest
	NodeID   string `form:"node_id"`
	RuleID   string `form:"rule_id"`
	Status   string `form:"status" binding:"omitempty,oneof=firing resolved acknowledged"`
	From     string `form:"from"` // RFC3339
	To       string `form:"to"`   // RFC3339
}

// AcknowledgeAlertEventRequest 确认告警事件请求
type AcknowledgeAlertEventRequest struct {
	AcknowledgedBy string `json:"acknowledged_by" binding:"required,min=1,max=255"`
}

// AlertEventResponse 告警事件响应
type AlertEventResponse struct {
	ID              string   `json:"id"`
	RuleID          string   `json:"rule_id"`
	NodeID          string   `json:"node_id"`
	MetricValue     float64  `json:"metric_value"`
	Status          string   `json:"status"`
	FiredAt         string   `json:"fired_at"`
	ResolvedAt      *string  `json:"resolved_at,omitempty"`
	AcknowledgedBy  *string  `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *string  `json:"acknowledged_at,omitempty"`
	NotifySent      bool     `json:"notify_sent"`
	NotifySentAt    *string  `json:"notify_sent_at,omitempty"`

	// Joined fields
	RuleName    string `json:"rule_name,omitempty"`
	NodeName    string `json:"node_name,omitempty"`
	MetricType  string `json:"metric_type,omitempty"`
	Operator    string `json:"operator,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
}
