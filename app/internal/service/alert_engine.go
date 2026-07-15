package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AlertEngine handles the evaluation of alert rules against metrics data
// and manages the lifecycle of alert events (fire/resolve/notify).
type AlertEngine struct {
	ruleRepo       *repository.AlertRuleRepository
	eventRepo      *repository.AlertEventRepository
	metricsRepo    *repository.EdgeNodeMetricsRepository
	nodeRepo       *repository.EdgeNodeRepository
	notifier       *NotifierRegistry
	// Map of ruleID+nodeID -> last notification time for silence period tracking.
	// Pre-populated from the database on startup to survive process restarts.
	silenceTracker sync.Map
}

// NewAlertEngine creates a new AlertEngine.
func NewAlertEngine(
	ruleRepo *repository.AlertRuleRepository,
	eventRepo *repository.AlertEventRepository,
	metricsRepo *repository.EdgeNodeMetricsRepository,
	nodeRepo *repository.EdgeNodeRepository,
	notifier *NotifierRegistry,
) *AlertEngine {
	return &AlertEngine{
		ruleRepo:    ruleRepo,
		eventRepo:   eventRepo,
		metricsRepo: metricsRepo,
		nodeRepo:    nodeRepo,
		notifier:    notifier,
	}
}

// silenceKey generates a unique key for silence tracking.
func (e *AlertEngine) silenceKey(ruleID, nodeID string) string {
	return ruleID + ":" + nodeID
}

// isInSilencePeriod checks if a rule is in its silence period for a node.
// Returns true if the last notification was sent within the silence window.
func (e *AlertEngine) isInSilencePeriod(ruleID, nodeID string, silenceMinutes int) bool {
	if silenceMinutes <= 0 {
		return false
	}

	key := e.silenceKey(ruleID, nodeID)
	val, ok := e.silenceTracker.Load(key)
	if !ok {
		return false
	}

	lastNotified, ok := val.(time.Time)
	if !ok {
		return false
	}

	return time.Since(lastNotified) < time.Duration(silenceMinutes)*time.Minute
}

// markNotified records that a notification was sent for a rule+node.
func (e *AlertEngine) markNotified(ruleID, nodeID string) {
	key := e.silenceKey(ruleID, nodeID)
	e.silenceTracker.Store(key, time.Now())
}

// RestoreSilenceState pre-populates the silence tracker from the database
// so that silence periods survive process restarts. Must be called once after
// NewAlertEngine, before any Evaluate* calls.
//
// Only restores entries whose NotifySentAt is still within the rule's
// SilenceMinutes window — entries that expired while the process was down
// are discarded so the next evaluate cycle fires notifications promptly.
func (e *AlertEngine) RestoreSilenceState(ctx context.Context) {
	records, err := e.eventRepo.FindFiringNotifySent(ctx)
	if err != nil {
		zap.L().Warn("alert engine: failed to restore silence state from database",
			zap.Error(err),
		)
		return
	}
	var restored int
	for _, r := range records {
		if r.NotifySentAt == nil {
			continue
		}
		// Fetch the rule to get SilenceMinutes; if not found, skip.
		rule, err := e.ruleRepo.FindByID(ctx, r.RuleID)
		if err != nil || rule == nil {
			continue
		}
		// Only restore if the silence period hasn't expired yet.
		if time.Since(*r.NotifySentAt) < time.Duration(rule.SilenceMinutes)*time.Minute {
			key := e.silenceKey(r.RuleID, r.NodeID)
			e.silenceTracker.Store(key, *r.NotifySentAt)
			restored++
		}
	}
	if restored > 0 {
		zap.L().Info("alert engine: restored silence state",
			zap.Int("event_count", restored),
		)
	}
}

// EvaluateAfterHeartbeat is called after a heartbeat has been processed and
// metrics persisted. It evaluates all applicable rules for the node and
// fires/resolves alert events as needed.
//
// This is the core method that should be called from HandleHeartbeat.
func (e *AlertEngine) EvaluateAfterHeartbeat(ctx context.Context, nodeID string, nodeName string, metricsRecord *model.EdgeNodeMetrics) error {
	// Get all active rules for this node
	rules, err := e.ruleRepo.ListActiveByNode(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("alert engine: list active rules: %w", err)
	}

	if len(rules) == 0 {
		return nil
	}

	for _, rule := range rules {
		// Get the metric value for this rule's metric type
		metricValue := e.extractMetricValueForRule(metricsRecord, rule.MetricType)

		if err := e.evaluateRule(ctx, &rule, nodeID, nodeName, metricValue); err != nil {
			zap.L().Error("alert engine: rule evaluation failed",
				zap.String("rule_id", rule.ID),
				zap.String("rule_name", rule.Name),
				zap.String("node_id", nodeID),
				zap.Error(err),
			)
			continue
		}
	}

	return nil
}

// evaluateRule checks a single rule against the current metric value and
// fires/resolves events based on the result.
func (e *AlertEngine) evaluateRule(ctx context.Context, rule *model.AlertRule, nodeID, nodeName string, currentValue float64) error {
	// Check if threshold is met
	thresholdMet := e.compareValue(currentValue, rule.Threshold, rule.Operator)

	// Get the latest firing event for this rule+node
	firingEvent, err := e.eventRepo.FindFiringByRuleAndNode(ctx, rule.ID, nodeID)
	if err != nil {
		return fmt.Errorf("find firing event: %w", err)
	}

	if thresholdMet {
		// Threshold is met - check duration requirement
		if rule.DurationSeconds > 0 {
			met, err := e.checkDuration(ctx, nodeID, rule)
			if err != nil {
				return fmt.Errorf("check duration: %w", err)
			}
			if !met {
				// Condition hasn't persisted long enough yet, don't fire
				return nil
			}
		}

		if firingEvent == nil {
			// No existing firing event - create a new one
			if err := e.fireEvent(ctx, rule, nodeID, currentValue); err != nil {
				return fmt.Errorf("fire event: %w", err)
			}
			// Re-fetch the newly created firing event for notification
			firingEvent, err = e.eventRepo.FindFiringByRuleAndNode(ctx, rule.ID, nodeID)
			if err != nil {
				return fmt.Errorf("find new firing event for notification: %w", err)
			}
			if firingEvent != nil {
				e.sendNotification(ctx, rule, nodeID, nodeName, currentValue, firingEvent)
			}
		} else {
			// Event already firing - check if we need to re-notify (silence period)
			var channels []string
			if err := json.Unmarshal(rule.NotifyChannels, &channels); err == nil && len(channels) > 0 {
				if !e.isInSilencePeriod(rule.ID, nodeID, rule.SilenceMinutes) {
					e.sendNotification(ctx, rule, nodeID, nodeName, currentValue, firingEvent)
				}
			}
		}
	} else {
		// Threshold not met
		if firingEvent != nil {
			// Event was firing - now resolved
			if err := e.resolveEvent(ctx, rule, nodeID, firingEvent); err != nil {
				return fmt.Errorf("resolve event: %w", err)
			}
		}
	}

	return nil
}

// fireEvent creates a new alert event and marks it as firing.
func (e *AlertEngine) fireEvent(ctx context.Context, rule *model.AlertRule, nodeID string, metricValue float64) error {
	event := &model.AlertEvent{
		BaseModel:   model.BaseModel{ID: uuid.New().String()},
		RuleID:      rule.ID,
		NodeID:      nodeID,
		MetricValue: metricValue,
		Status:      model.AlertEventStatusFiring,
		FiredAt:     time.Now(),
	}

	if err := e.eventRepo.Create(ctx, event); err != nil {
		return fmt.Errorf("create alert event: %w", err)
	}

	zap.L().Warn("alert event fired",
		zap.String("event_id", event.ID),
		zap.String("rule_id", rule.ID),
		zap.String("rule_name", rule.Name),
		zap.String("node_id", nodeID),
		zap.Float64("metric_value", metricValue),
		zap.Float64("threshold", rule.Threshold),
	)

	return nil
}

// resolveEvent resolves a firing alert event.
func (e *AlertEngine) resolveEvent(ctx context.Context, rule *model.AlertRule, nodeID string, event *model.AlertEvent) error {
	if err := e.eventRepo.ResolveEvent(ctx, event.ID); err != nil {
		return fmt.Errorf("resolve event: %w", err)
	}

	zap.L().Info("alert event resolved",
		zap.String("event_id", event.ID),
		zap.String("rule_id", rule.ID),
		zap.String("node_id", nodeID),
	)

	return nil
}

// sendNotification sends a notification through the rule's configured channels.
// firingEvent must be a non-nil firing event that triggered or is triggering this notification.
func (e *AlertEngine) sendNotification(ctx context.Context, rule *model.AlertRule, nodeID, nodeName string, metricValue float64, firingEvent *model.AlertEvent) {
	// firingEvent is expected to be non-nil when called; guard anyway.
	if firingEvent == nil {
		zap.L().Warn("alert engine: nil firing event passed to sendNotification",
			zap.String("rule_id", rule.ID),
			zap.String("node_id", nodeID),
		)
		return
	}

	var channels []string
	if err := json.Unmarshal(rule.NotifyChannels, &channels); err != nil || len(channels) == 0 {
		// Default to all configured notifiers if no specific channels set
		channels = []string{"webhook", "email", "telegram"}
	}

	// Mark notify as sent
	if err := e.eventRepo.MarkNotifySent(ctx, firingEvent.ID); err != nil {
		zap.L().Error("alert engine: mark notify sent failed",
			zap.String("event_id", firingEvent.ID),
			zap.Error(err),
		)
	}

	// Send through all configured notifiers
	e.notifier.SendToChannels(ctx, channels, firingEvent, rule, nodeName)

	// Track silence period
	e.markNotified(rule.ID, nodeID)
}

// checkDuration checks if the threshold condition has been met for the required duration.
// It queries the metrics table for records strictly BEFORE now (excluding the just-inserted
// record) to prevent the current heartbeat from trivially satisfying the check when the
// duration window contains only a single data point.
func (e *AlertEngine) checkDuration(ctx context.Context, nodeID string, rule *model.AlertRule) (bool, error) {
	if rule.DurationSeconds <= 0 {
		return true, nil
	}

	cutoff := time.Now().Add(-time.Duration(rule.DurationSeconds) * time.Second)
	// Exclude the record that was just inserted in THIS heartbeat cycle.
	// Without this offset, a single freshly-inserted record trivially satisfies
	// the "all records in window violate" check when DurationSeconds is small.
	endWindow := time.Now().Add(-time.Second)

	// Count total records in the window (excluding the current heartbeat)
	total, err := e.metricsRepo.CountByNode(ctx, nodeID, cutoff, endWindow)
	if err != nil {
		return false, fmt.Errorf("count metrics for duration check: %w", err)
	}

	if total == 0 {
		return false, nil
	}

	// For duration check, we need to verify the persisted metrics
	// We query the column mapped to the metric type
	column, ok := metricColumnMapForAlert[rule.MetricType]
	if !ok {
		return false, fmt.Errorf("unknown metric type: %s", rule.MetricType)
	}

	violationCount, err := e.metricsRepo.CountViolationsInWindow(ctx, nodeID, column, rule.Operator, rule.Threshold, cutoff, endWindow)
	if err != nil {
		return false, fmt.Errorf("check duration violation: %w", err)
	}

	// All records in the window must meet the threshold
	return violationCount == total, nil
}

// extractMetricValueForRule extracts the current metric value for a specific rule's metric type.
func (e *AlertEngine) extractMetricValueForRule(metrics *model.EdgeNodeMetrics, metricType string) float64 {
	if metrics == nil {
		return 0
	}

	// Handle status-based metric types separately
	switch metricType {
	case "node_offline", "node_error":
		// When we have a metrics record, the node is online.
		// For status rules, 0 = normal (online/not error).
		return 0
	default:
		val, ok := extractMetricValueForRule(metrics, metricType)
		if !ok {
			return 0
		}
		return val
	}
}

// compareValue compares a value against a threshold using the given operator.
func (e *AlertEngine) compareValue(value, threshold float64, operator string) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	default:
		return false
	}
}

// metricColumnMapForAlert maps metric types to the database column name for alert evaluation
var metricColumnMapForAlert = map[string]string{
	"cpu_usage":         "cpu_usage",
	"cpu_load_1m":       "cpu_load_1m",
	"cpu_load_5m":       "cpu_load_5m",
	"cpu_load_15m":      "cpu_load_15m",
	"memory_usage":      "memory_usage",
	"temperature":       "temperature",
	"process_count":     "process_count",
	"thread_count":      "thread_count",
	"uptime":            "uptime",
	"net_rx_speed":      "net_rx_speed",
	"net_tx_speed":      "net_tx_speed",
	"worker_count":      "worker_count",
	"active_stream_count": "active_stream_count",
	"current_load":      "current_load",
}

// extractMetricValueForRule extracts the value for a specific metric type from the metrics record.
func extractMetricValueForRule(metrics *model.EdgeNodeMetrics, metricType string) (float64, bool) {
	if metrics == nil {
		return 0, false
	}

	switch metricType {
	case "cpu_usage":
		return metrics.CPUUsage, true
	case "memory_usage":
		return metrics.MemoryUsage, true
	case "disk_usage":
		// For disk_usage, return the maximum usage across all mount points
		return extractMaxDiskUsage(metrics), true
	case "temperature":
		return metrics.Temperature, true
	case "node_offline":
		// Derived status - always evaluate as 0 when we have metrics (node is online)
		return 0, true
	case "node_error":
		// Derived status - always evaluate as 0 when we have a healthy metrics record
		return 0, true
	default:
		return 0, false
	}
}

// extractMaxDiskUsage returns the maximum disk usage percentage from the JSON array.
func extractMaxDiskUsage(metrics *model.EdgeNodeMetrics) float64 {
	if len(metrics.DiskUsage) == 0 {
		return 0
	}

	var diskInfos []struct {
		Path    string  `json:"path"`
		Percent float64 `json:"percent"`
	}
	if err := json.Unmarshal(metrics.DiskUsage, &diskInfos); err != nil {
		zap.L().Warn("alert engine: failed to unmarshal disk_usage for metric extraction",
			zap.String("metric_type", "disk_usage"),
			zap.Error(err),
		)
		return 0
	}

	var maxPercent float64
	for _, di := range diskInfos {
		if di.Percent > maxPercent {
			maxPercent = di.Percent
		}
	}
	return maxPercent
}

// EvaluateNodeOffline checks if a node should trigger offline/error alert rules.
// Called when a node goes offline or reports an error status.
func (e *AlertEngine) EvaluateNodeOffline(ctx context.Context, nodeID string, status string) error {
	metricType := "node_offline"
	if status == model.NodeStatusError {
		metricType = "node_error"
	}

	rules, err := e.ruleRepo.ListActiveByNode(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("alert engine: list active rules: %w", err)
	}

	node, err := e.nodeRepo.FindByID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("alert engine: find node: %w", err)
	}

	for _, rule := range rules {
		if rule.MetricType != metricType {
			continue
		}

		// For offline/error rules, the metric value is 1 when the condition is true
		var metricValue float64 = 1

		firingEvent, err := e.eventRepo.FindFiringByRuleAndNode(ctx, rule.ID, nodeID)
		if err != nil {
			zap.L().Error("alert engine: find firing event", zap.Error(err))
			continue
		}

		if firingEvent == nil {
			if err := e.fireEvent(ctx, &rule, nodeID, metricValue); err != nil {
				zap.L().Error("alert engine: fire event", zap.Error(err))
				continue
			}
				// Re-fetch the newly created firing event for notification
				newEvent, fetchErr := e.eventRepo.FindFiringByRuleAndNode(ctx, rule.ID, nodeID)
				if fetchErr != nil {
					zap.L().Error("alert engine: find new firing event for notification", zap.Error(fetchErr))
					continue
				}
				if newEvent != nil {
					e.sendNotification(ctx, &rule, nodeID, node.Name, metricValue, newEvent)
				}
		}
	}

	return nil
}

// EvaluateNodeBackOnline checks if offline/error alert rules should be resolved
// when a node comes back online or recovers from error.
func (e *AlertEngine) EvaluateNodeBackOnline(ctx context.Context, nodeID string) error {
	metricTypes := []string{"node_offline", "node_error"}

	rules, err := e.ruleRepo.ListActiveByNode(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("alert engine: list active rules: %w", err)
	}

	for _, rule := range rules {
		isStatusRule := false
		for _, mt := range metricTypes {
			if rule.MetricType == mt {
				isStatusRule = true
				break
			}
		}
		if !isStatusRule {
			continue
		}

		firingEvent, err := e.eventRepo.FindFiringByRuleAndNode(ctx, rule.ID, nodeID)
		if err != nil {
			zap.L().Error("alert engine: find firing event", zap.Error(err))
			continue
		}

		if firingEvent != nil {
			if err := e.resolveEvent(ctx, &rule, nodeID, firingEvent); err != nil {
				zap.L().Error("alert engine: resolve event", zap.Error(err))
			}
		}
	}

	return nil
}

// isStatusRule checks if a metric type represents a node status (offline/error).
func isStatusMetric(metricType string) bool {
	return metricType == "node_offline" || metricType == "node_error"
}

// CountActiveAlerts returns the total number of firing alert events.
func (e *AlertEngine) CountActiveAlerts(ctx context.Context) (int64, error) {
	return e.eventRepo.CountFiringTotal(ctx)
}
