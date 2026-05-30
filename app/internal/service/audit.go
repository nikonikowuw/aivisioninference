// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"context"
	"strconv"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/csvx"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/i18n"
	"github.com/niko-admin/niko-admin/internal/pkg/timex"
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

// List 查询审计日志列表，支持关键字、操作类型、资源类型、时间范围等筛选条件。
func (s *AuditService) List(ctx context.Context, lang string, req dto.ListAuditLogRequest) (*ListResult, error) {
	if err := normalizeAuditTimeRange(&req); err != nil {
		return nil, err
	}

	auditLogs, total, err := s.auditRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	list := make([]dto.AuditLogResponse, len(auditLogs))
	for i, log := range auditLogs {
		list[i] = dto.AuditLogResponse{
			ID:                 log.ID,
			UserID:             log.UserID,
			Username:           log.Username,
			ActionType:         log.ActionType,
			ActionTypeLabel:    i18n.TranslateAction(lang, log.ActionType),
			ResourceType:       log.ResourceType,
			ResourceTypeLabel:  i18n.TranslateResource(lang, log.ResourceType),
			ResourceID:         log.ResourceID,
			RequestPath:        log.RequestPath,
			RequestMethod:      log.RequestMethod,
			RequestMethodLabel: i18n.TranslateMethod(lang, log.RequestMethod),
			RequestIP:          log.RequestIP,
			UserAgent:          log.UserAgent,
			ResponseStatus:     log.ResponseStatus,
			DurationMs:         log.DurationMs,
			ResultSummary:      log.ResultSummary,
			ResultSummaryLabel: i18n.TranslateResult(lang, log.ResultSummary),
			CreatedAt:          log.CreatedAt,
		}
	}

	return &ListResult{List: list, Total: total}, nil
}

// Create 插入一条新的审计日志记录
func (s *AuditService) Create(ctx context.Context, log *model.AuditLog) error {
	return s.auditRepo.Create(ctx, log)
}

// ExportCSV 导出当前筛选条件下的审计日志 CSV。
func (s *AuditService) ExportCSV(ctx context.Context, lang string, req dto.ListAuditLogRequest) ([]byte, error) {
	if err := normalizeAuditTimeRange(&req); err != nil {
		return nil, err
	}
	logs, err := s.auditRepo.ListForExport(ctx, req, maxCSVExportRows)
	if err != nil {
		zap.L().Error("export audit logs failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	rows := make([][]string, 0, len(logs))
	for _, log := range logs {
		rows = append(rows, []string{
			log.ID,
			log.Username,
			i18n.TranslateAction(lang, log.ActionType),
			i18n.TranslateResource(lang, log.ResourceType),
			log.RequestMethod,
			log.RequestPath,
			strconv.Itoa(log.ResponseStatus),
			strconv.FormatInt(log.DurationMs, 10),
			i18n.TranslateResult(lang, log.ResultSummary),
			log.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	data, err := csvx.Build([]string{"ID", "Username", "Action", "Resource", "Method", "Path", "Status", "DurationMs", "Result", "CreatedAt"}, rows)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return data, nil
}

func normalizeAuditTimeRange(req *dto.ListAuditLogRequest) error {
	return timex.NormalizeRange(req.StartTime, req.EndTime, &req.FromTime, &req.ToTime)
}
