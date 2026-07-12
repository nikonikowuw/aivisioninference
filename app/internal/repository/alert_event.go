package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// AlertEventRepository handles database operations for AlertEvent model.
type AlertEventRepository struct {
	db *gorm.DB
}

// NewAlertEventRepository creates a new AlertEventRepository.
func NewAlertEventRepository(db *gorm.DB) *AlertEventRepository {
	return &AlertEventRepository{db: db}
}

// Create inserts a new alert event.
func (r *AlertEventRepository) Create(ctx context.Context, event *model.AlertEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// FindByID finds an alert event by its ID.
func (r *AlertEventRepository) FindByID(ctx context.Context, id string) (*model.AlertEvent, error) {
	var event model.AlertEvent
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// Update saves changes to an alert event.
func (r *AlertEventRepository) Update(ctx context.Context, event *model.AlertEvent) error {
	return r.db.WithContext(ctx).Save(event).Error
}

// List returns a paginated list of alert events with optional filters.
// Supports joining with alert_rules and edge_nodes for display.
func (r *AlertEventRepository) List(ctx context.Context, req dto.AlertEventListRequest) ([]dto.AlertEventResponse, int64, error) {
	var total int64
	var results []dto.AlertEventResponse

	db := r.db.WithContext(ctx)

	// Count query (model-based, no joins needed)
	countQuery := db.Model(&model.AlertEvent{})
	if req.NodeID != "" {
		countQuery = countQuery.Where("node_id = ?", req.NodeID)
	}
	if req.RuleID != "" {
		countQuery = countQuery.Where("rule_id = ?", req.RuleID)
	}
	if req.Status != "" {
		countQuery = countQuery.Where("status = ?", req.Status)
	}
	if req.From != "" {
		from, err := time.Parse(time.RFC3339, req.From)
		if err == nil {
			countQuery = countQuery.Where("fired_at >= ?", from)
		}
	}
	if req.To != "" {
		to, err := time.Parse(time.RFC3339, req.To)
		if err == nil {
			countQuery = countQuery.Where("fired_at <= ?", to)
		}
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Data query with joined fields
	selectFields := `alert_events.id, alert_events.rule_id, alert_events.node_id,
		alert_events.metric_value, alert_events.status,
		alert_events.fired_at, alert_events.resolved_at,
		alert_events.acknowledged_by, alert_events.acknowledged_at,
		alert_events.notify_sent, alert_events.notify_sent_at,
		COALESCE(alert_rules.name, '') as rule_name,
		COALESCE(alert_rules.metric_type, '') as metric_type,
		COALESCE(alert_rules.operator, '') as operator,
		COALESCE(alert_rules.threshold, 0) as threshold,
		COALESCE(edge_nodes.name, '') as node_name`

	dataQuery := db.Table("alert_events").
		Select(selectFields).
		Joins("LEFT JOIN alert_rules ON alert_events.rule_id = alert_rules.id").
		Joins("LEFT JOIN edge_nodes ON alert_events.node_id = edge_nodes.id")

	if req.NodeID != "" {
		dataQuery = dataQuery.Where("alert_events.node_id = ?", req.NodeID)
	}
	if req.RuleID != "" {
		dataQuery = dataQuery.Where("alert_events.rule_id = ?", req.RuleID)
	}
	if req.Status != "" {
		dataQuery = dataQuery.Where("alert_events.status = ?", req.Status)
	}
	if req.From != "" {
		from, err := time.Parse(time.RFC3339, req.From)
		if err == nil {
			dataQuery = dataQuery.Where("alert_events.fired_at >= ?", from)
		}
	}
	if req.To != "" {
		to, err := time.Parse(time.RFC3339, req.To)
		if err == nil {
			dataQuery = dataQuery.Where("alert_events.fired_at <= ?", to)
		}
	}

	err := dataQuery.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderByDefault(),
	).Order("alert_events.fired_at DESC").
		Find(&results).Error
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// FindFiringByRuleAndNode finds the latest firing event for a specific rule and node.
func (r *AlertEventRepository) FindFiringByRuleAndNode(ctx context.Context, ruleID, nodeID string) (*model.AlertEvent, error) {
	var event model.AlertEvent
	err := r.db.WithContext(ctx).
		Where("rule_id = ? AND node_id = ? AND status = ?",
			ruleID, nodeID, model.AlertEventStatusFiring).
		Order("fired_at DESC").
		First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

// FindLatestEventByRuleAndNode finds the latest event (any status) for a specific rule and node.
func (r *AlertEventRepository) FindLatestEventByRuleAndNode(ctx context.Context, ruleID, nodeID string) (*model.AlertEvent, error) {
	var event model.AlertEvent
	err := r.db.WithContext(ctx).
		Where("rule_id = ? AND node_id = ?", ruleID, nodeID).
		Order("fired_at DESC").
		First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

// AcknowledgeEvent marks an alert event as acknowledged.
func (r *AlertEventRepository) AcknowledgeEvent(ctx context.Context, id string, acknowledgedBy string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("id = ?", id).
		Where("status IN ?", []string{model.AlertEventStatusFiring, model.AlertEventStatusResolved}).
		Updates(map[string]interface{}{
			"status":           model.AlertEventStatusAcknowledged,
			"acknowledged_by":  acknowledgedBy,
			"acknowledged_at":  &now,
		}).Error
}

// ResolveEvent marks a firing event as resolved.
func (r *AlertEventRepository) ResolveEvent(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("id = ? AND status = ?", id, model.AlertEventStatusFiring).
		Updates(map[string]interface{}{
			"status":      model.AlertEventStatusResolved,
			"resolved_at": &now,
		}).Error
}

// MarkNotifySent marks an event as having been notified.
func (r *AlertEventRepository) MarkNotifySent(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"notify_sent":    true,
			"notify_sent_at": &now,
		}).Error
}

// CountFiringByNode counts active firing events for a node.
func (r *AlertEventRepository) CountFiringByNode(ctx context.Context, nodeID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("node_id = ? AND status = ?", nodeID, model.AlertEventStatusFiring).
		Count(&count).Error
	return count, err
}

// CountFiringTotal returns the total number of firing events across all nodes.
func (r *AlertEventRepository) CountFiringTotal(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("status = ?", model.AlertEventStatusFiring).
		Count(&count).Error
	return count, err
}

// FindFiringNotifySent returns all firing events that have had a notification sent,
// along with the time it was sent. Used to restore silence state across restarts.
func (r *AlertEventRepository) FindFiringNotifySent(ctx context.Context) ([]model.AlertEvent, error) {
	var events []model.AlertEvent
	err := r.db.WithContext(ctx).
		Where("status = ? AND notify_sent = ? AND notify_sent_at IS NOT NULL",
			model.AlertEventStatusFiring, true).
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

// WithTx returns a repository bound to the provided transaction.
func (r *AlertEventRepository) WithTx(tx *gorm.DB) *AlertEventRepository {
	return &AlertEventRepository{db: tx}
}

// DB returns the underlying GORM DB instance.
func (r *AlertEventRepository) DB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}
