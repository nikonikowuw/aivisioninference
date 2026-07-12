package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// TerminalSessionRepository 终端会话数据库操作
type TerminalSessionRepository struct {
	db *gorm.DB
}

// NewTerminalSessionRepository 创建 TerminalSessionRepository
func NewTerminalSessionRepository(db *gorm.DB) *TerminalSessionRepository {
	return &TerminalSessionRepository{db: db}
}

// Create 创建终端会话
func (r *TerminalSessionRepository) Create(ctx context.Context, session *model.TerminalSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// FindByID 根据 ID 查询会话
func (r *TerminalSessionRepository) FindByID(ctx context.Context, id string) (*model.TerminalSession, error) {
	var session model.TerminalSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error
	return &session, err
}

// Update 更新会话
func (r *TerminalSessionRepository) Update(ctx context.Context, session *model.TerminalSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

// UpdateFields 更新指定字段
func (r *TerminalSessionRepository) UpdateFields(ctx context.Context, id string, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.TerminalSession{}).Where("id = ?", id).Updates(fields).Error
}

// ListByNodeID 按节点 ID 分页查询会话（不返回录制数据）
func (r *TerminalSessionRepository) ListByNodeID(ctx context.Context, nodeID string, status string, page, pageSize int, sort, order string) ([]model.TerminalSession, int64, error) {
	var items []model.TerminalSession
	var total int64

	query := r.db.WithContext(ctx).Model(&model.TerminalSession{}).Where("node_id = ?", nodeID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Omit("recording_data").Scopes(
		scopes.Paginate(page, pageSize),
		scopes.OrderBy(sort, order, model.TerminalSession{}.SortableFields()...),
	).Find(&items).Error
	return items, total, err
}

// ListPausedBefore 查询暂停超过指定时长的会话
func (r *TerminalSessionRepository) ListPausedBefore(ctx context.Context, timeout time.Duration) ([]model.TerminalSession, error) {
	var items []model.TerminalSession
	cutoff := time.Now().Add(-timeout)
	err := r.db.WithContext(ctx).
		Where("status = ? AND paused_at IS NOT NULL AND paused_at < ?", model.TerminalSessionStatusPaused, cutoff).
		Find(&items).Error
	return items, err
}

// CountActiveByNodeID 查询指定节点活跃会话数
func (r *TerminalSessionRepository) CountActiveByNodeID(ctx context.Context, nodeID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.TerminalSession{}).
		Where("node_id = ? AND status IN ?", nodeID, []string{model.TerminalSessionStatusActive, model.TerminalSessionStatusPaused}).
		Count(&count).Error
	return count, err
}
