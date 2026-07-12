package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// AlertRuleRepository handles database operations for AlertRule model.
type AlertRuleRepository struct {
	db *gorm.DB
}

// NewAlertRuleRepository creates a new AlertRuleRepository.
func NewAlertRuleRepository(db *gorm.DB) *AlertRuleRepository {
	return &AlertRuleRepository{db: db}
}

// Create inserts a new alert rule.
func (r *AlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

// FindByID finds an alert rule by its ID.
func (r *AlertRuleRepository) FindByID(ctx context.Context, id string) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// Update saves changes to an alert rule.
func (r *AlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

// Delete soft-deletes an alert rule by ID.
func (r *AlertRuleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.AlertRule{}, "id = ?", id).Error
}

// List returns a paginated list of alert rules with optional filters.
func (r *AlertRuleRepository) List(ctx context.Context, req dto.AlertRuleListRequest) ([]model.AlertRule, int64, error) {
	var items []model.AlertRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AlertRule{})

	if req.Keyword != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}
	if req.NodeID != "" {
		query = query.Where("node_id = ? OR node_id IS NULL", req.NodeID)
	}
	if req.MetricType != "" {
		query = query.Where("metric_type = ?", req.MetricType)
	}
	if req.Enabled == "true" {
		query = query.Where("enabled = ?", true)
	} else if req.Enabled == "false" {
		query = query.Where("enabled = ?", false)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderBy(req.Sort, req.Order, model.AlertRule{}.SortableFields()...),
	).Find(&items).Error
	return items, total, err
}

// ListActiveByNode returns all enabled rules that apply to a specific node.
// Returns both node-specific rules and global rules (node_id IS NULL).
func (r *AlertRuleRepository) ListActiveByNode(ctx context.Context, nodeID string) ([]model.AlertRule, error) {
	var rules []model.AlertRule
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Where("node_id = ? OR node_id IS NULL", nodeID).
		Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// GetByName returns a rule by name for duplicate checking.
func (r *AlertRuleRepository) GetByName(ctx context.Context, name string, excludeID string) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.AlertRule{}).Where("name = ?", name)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByName checks if a rule name already exists (excluding a specific ID).
func (r *AlertRuleRepository) ExistsByName(ctx context.Context, name, excludeID string) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.AlertRule{}).Where("name = ?", name)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindByNodeID retrieves all rules for a specific node (including global rules).
func (r *AlertRuleRepository) FindByNodeID(ctx context.Context, nodeID string) ([]model.AlertRule, error) {
	var rules []model.AlertRule
	err := r.db.WithContext(ctx).
		Where("node_id = ? OR node_id IS NULL", nodeID).
		Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// WithTx returns a repository bound to the provided transaction.
func (r *AlertRuleRepository) WithTx(tx *gorm.DB) *AlertRuleRepository {
	return &AlertRuleRepository{db: tx}
}

// DB returns the underlying GORM DB instance.
func (r *AlertRuleRepository) DB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// FindByIDForUpdate finds an alert rule by ID with pessimistic lock.
func (r *AlertRuleRepository) FindByIDForUpdate(ctx context.Context, id string) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}
