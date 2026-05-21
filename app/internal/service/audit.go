package service

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/i18n"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AuditService 处理审计日志相关的业务逻辑
type AuditService struct {
	auditRepo *repository.AuditRepository // 审计日志数据持久化接口
}

// NewAuditService 创建并返回一个新的 AuditService 实例
func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

// ListResult 包含分页查询的审计日志响应结果
type ListResult struct {
	List  []dto.AuditLogResponse // 审计日志响应对象列表
	Total int64                  // 满足条件的总记录数
}

// List 根据分页和可选的过滤条件返回审计日志列表和总条数
//
// 【核心功能】查询审计日志列表，支持关键字、操作类型、资源类型、时间范围等筛选条件。
// 时间范围参数通过公共 ParseTimeRange 方法解析，确保格式统一。
func (s *AuditService) List(ctx context.Context, lang string, req dto.ListAuditLogRequest) (*ListResult, error) {
	// 解析时间范围参数
	if req.StartTime != "" || req.EndTime != "" {
		from, to, err := scopes.ParseTimeRange(req.StartTime, req.EndTime)
		if err != nil {
			return nil, mapTimeRangeError(err)
		}
		req.FromTime = from
		req.ToTime = to
	}

	logs, total, err := s.auditRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	list := make([]dto.AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		list = append(list, dto.AuditLogResponse{
			ID:             l.ID,
			UserID:         l.UserID,
			Username:       l.Username,
			ActionType:     i18n.TranslateAction(lang, l.ActionType),
			ResourceType:   l.ResourceType,
			ResourceID:     l.ResourceID,
			RequestPath:    l.RequestPath,
			RequestMethod:  l.RequestMethod,
			RequestIP:      l.RequestIP,
			UserAgent:      l.UserAgent,
			ResponseStatus: l.ResponseStatus,
			DurationMs:     l.DurationMs,
			ResultSummary:  l.ResultSummary,
			CreatedAt:      l.CreatedAt,
		})
	}

	return &ListResult{List: list, Total: total}, nil
}

// Create 插入一条新的审计日志记录
func (s *AuditService) Create(ctx context.Context, log *model.AuditLog) error {
	return s.auditRepo.Create(ctx, log)
}
