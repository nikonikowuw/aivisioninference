package service

import (
	"context"
	"fmt"
	"time"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
	"go.uber.org/zap"
)

// EdgeNodeMetricsService handles business logic for EdgeNodeMetrics operations.
type EdgeNodeMetricsService struct {
	metricsRepo *repository.EdgeNodeMetricsRepository
}

// NewEdgeNodeMetricsService creates a new EdgeNodeMetricsService.
func NewEdgeNodeMetricsService(metricsRepo *repository.EdgeNodeMetricsRepository) *EdgeNodeMetricsService {
	return &EdgeNodeMetricsService{
		metricsRepo: metricsRepo,
	}
}

// RecordMetrics 插入一次指标记录。
func (s *EdgeNodeMetricsService) RecordMetrics(ctx context.Context, metrics *model.EdgeNodeMetrics) error {
	if metrics.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}
	if metrics.CreatedAt.IsZero() {
		metrics.CreatedAt = time.Now()
	}

	if err := s.metricsRepo.Create(ctx, metrics); err != nil {
		zap.L().Error("failed to record edge node metrics",
			zap.String("node_id", metrics.NodeID),
			zap.Error(err),
		)
		return fmt.Errorf("记录指标失败: %w", err)
	}
	return nil
}

// QueryMetrics 查询指定节点的历史指标数据，支持按时间范围、指标类型、聚合方式和分页。
func (s *EdgeNodeMetricsService) QueryMetrics(ctx context.Context, nodeID string, req dto.MetricQueryRequest) (*dto.MetricQueryResponse, error) {
	items, total, err := s.metricsRepo.ListMetrics(ctx, nodeID, req)
	if err != nil {
		return nil, fmt.Errorf("查询指标失败: %w", err)
	}

	return &dto.MetricQueryResponse{
		List:  items,
		Total: total,
	}, nil
}

// DeleteOldMetrics 清理早于指定时间的指标记录。
// 由定时任务调度，默认保留 7 天数据。
func (s *EdgeNodeMetricsService) DeleteOldMetrics(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	count, err := s.metricsRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		zap.L().Error("failed to delete old edge node metrics",
			zap.Time("cutoff", cutoff),
			zap.Error(err),
		)
		return 0, fmt.Errorf("清理过期指标失败: %w", err)
	}

	if count > 0 {
		zap.L().Info("deleted old edge node metrics",
			zap.Int64("count", count),
			zap.Time("cutoff", cutoff),
		)
	}

	return count, nil
}
