// Package repository 提供数据访问层实现，封装 GORM 数据库操作。
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// AuditRepository 处理 AuditLog 审计日志模型的数据持久化操作
type AuditRepository struct {
	db *gorm.DB // GORM 数据库连接实例
}

// NewAuditRepository 创建并返回一个新的 AuditRepository 实例
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create 往数据库中插入一条新的审计日志记录
func (r *AuditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// List 分页查询并返回满足筛选条件的审计日志列表及总条数
func (r *AuditRepository) List(ctx context.Context, req dto.ListAuditLogRequest) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64

	// 应用请求中指定的各种查询范围 scopes (过滤条件)
	query := r.db.WithContext(ctx).Model(&model.AuditLog{}).Scopes(req.FilterScopes()...)

	// 统计符合过滤条件的总条数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序和分页查询数据
	err := query.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderBy(req.Sort, req.Order, model.AuditLog{}.SortableFields()...),
	).Find(&logs).Error
	return logs, total, err
}

// ListForExport 返回符合筛选条件的审计日志列表，用于导出。
func (r *AuditRepository) ListForExport(ctx context.Context, req dto.ListAuditLogRequest, limit int) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := r.db.WithContext(ctx).
		Model(&model.AuditLog{}).
		Scopes(req.FilterScopes()...).
		Scopes(scopes.OrderBy(req.Sort, req.Order, model.AuditLog{}.SortableFields()...), scopes.OrderByDefault()).
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
