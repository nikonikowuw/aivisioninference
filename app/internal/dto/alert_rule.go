package dto

// AlertRuleListRequest 告警规则列表查询参数
type AlertRuleListRequest struct {
	PageRequest
	Keyword    string `form:"keyword"`
	NodeID     string `form:"node_id"`
	MetricType string `form:"metric_type" binding:"omitempty,oneof=cpu_usage memory_usage disk_usage node_offline node_error temperature"`
	Enabled    string `form:"enabled"` // "true" or "false"
}

// CreateAlertRuleRequest 创建告警规则请求
type CreateAlertRuleRequest struct {
	Name             string   `json:"name" binding:"required,min=1,max=255"`
	NodeID           *string  `json:"node_id,omitempty"` // nil = global rule
	MetricType       string   `json:"metric_type" binding:"required,oneof=cpu_usage memory_usage disk_usage node_offline node_error temperature"`
	Operator         string   `json:"operator" binding:"required,oneof='>' '>=' '<' '<=' '=='"`
	Threshold        float64  `json:"threshold" binding:"required"`
	DurationSeconds  int      `json:"duration_seconds" binding:"min=0"`
	SilenceMinutes   int      `json:"silence_minutes" binding:"min=0"`
	Enabled          *bool    `json:"enabled,omitempty"`
	NotifyChannels   []string `json:"notify_channels,omitempty"`
	Description      string   `json:"description" binding:"max=1000"`
}

// UpdateAlertRuleRequest 更新告警规则请求
type UpdateAlertRuleRequest struct {
	Name             *string  `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	NodeID           *string  `json:"node_id,omitempty"`
	MetricType       *string  `json:"metric_type,omitempty" binding:"omitempty,oneof=cpu_usage memory_usage disk_usage node_offline node_error temperature"`
	Operator         *string  `json:"operator,omitempty" binding:"omitempty,oneof='>' '>=' '<' '<=' '=='"`
	Threshold        *float64 `json:"threshold,omitempty"`
	DurationSeconds  *int     `json:"duration_seconds,omitempty" binding:"omitempty,min=0"`
	SilenceMinutes   *int     `json:"silence_minutes,omitempty" binding:"omitempty,min=0"`
	Enabled          *bool    `json:"enabled,omitempty"`
	NotifyChannels   []string `json:"notify_channels,omitempty"`
	Description      *string  `json:"description,omitempty" binding:"omitempty,max=1000"`
}
