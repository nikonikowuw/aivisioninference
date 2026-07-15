package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
	"gorm.io/gorm"
)

// EdgeNodeEngineMetricsRepository handles database operations for EdgeNodeEngineMetrics model.
type EdgeNodeEngineMetricsRepository struct {
	db *gorm.DB
}

// NewEdgeNodeEngineMetricsRepository creates a new EdgeNodeEngineMetricsRepository.
func NewEdgeNodeEngineMetricsRepository(db *gorm.DB) *EdgeNodeEngineMetricsRepository {
	return &EdgeNodeEngineMetricsRepository{db: db}
}

// Create inserts a new metrics record.
func (r *EdgeNodeEngineMetricsRepository) Create(ctx context.Context, item *model.EdgeNodeEngineMetrics) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// engineMetricColumnMap maps metric type names to their database column names
var engineMetricColumnMap = map[string]string{
	"active_stream_count":      "active_stream_count",
	"dma_used_bytes":           "dma_used_bytes",
	"dma_total_bytes":          "dma_total_bytes",
	"npu_used_bytes":           "npu_used_bytes",
	"npu_total_bytes":          "npu_total_bytes",
	"worker_count":             "worker_count",
	"idle_worker_count":        "idle_worker_count",
	"decode_sessions":          "decode_sessions",
	"encode_sessions":          "encode_sessions",
	"decode_slots_used":        "decode_slots_used",
	"encode_slots_used":        "encode_slots_used",
	"egress_bps":               "egress_bps",
	"preview_pipeline_count":   "preview_pipeline_count",
	"inference_pipeline_count": "inference_pipeline_count",
	"mixed_pipeline_count":     "mixed_pipeline_count",
	"preview_capacity":         "preview_capacity",
	"preview_in_use":           "preview_in_use",
	"accelerator_utilization":  "accelerator_utilization",
}

// ListMetrics returns paginated time-series metrics for a specific node and metric type.
func (r *EdgeNodeEngineMetricsRepository) ListMetrics(ctx context.Context, nodeID string, req dto.MetricQueryRequest) ([]dto.MetricDataPoint, int64, error) {
	from, err := req.GetFromTime()
	if err != nil {
		return nil, 0, fmt.Errorf("invalid from time: %w", err)
	}
	to, err := req.GetToTime()
	if err != nil {
		return nil, 0, fmt.Errorf("invalid to time: %w", err)
	}

	column, ok := engineMetricColumnMap[req.Metric]
	if !ok {
		return nil, 0, fmt.Errorf("unsupported engine metric type: %s", req.Metric)
	}

	if req.Aggregation != "" && req.Interval != "" {
		return r.listMetricsAggregated(ctx, nodeID, column, from, to, req)
	}

	return r.listMetricsRaw(ctx, nodeID, column, from, to, req)
}

func (r *EdgeNodeEngineMetricsRepository) listMetricsRaw(ctx context.Context, nodeID string, column string, from, to time.Time, req dto.MetricQueryRequest) ([]dto.MetricDataPoint, int64, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).Model(&model.EdgeNodeEngineMetrics{}).
		Where("node_id = ?", nodeID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to)

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf("created_at as t, %s as v", column)

	var results []dto.MetricDataPoint
	err := r.db.WithContext(ctx).Model(&model.EdgeNodeEngineMetrics{}).
		Select(query).
		Where("node_id = ?", nodeID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Scopes(
			scopes.Paginate(req.GetPage(), req.GetPageSize()),
		).
		Order("created_at ASC").
		Find(&results).Error
	if err != nil {
		return nil, 0, err
	}

	for i := range results {
		if t, ok := parseTimestamp(results[i].T); ok {
			results[i].T = t.Format(time.RFC3339)
		}
	}

	return results, total, nil
}

func (r *EdgeNodeEngineMetricsRepository) listMetricsAggregated(ctx context.Context, nodeID string, column string, from, to time.Time, req dto.MetricQueryRequest) ([]dto.MetricDataPoint, int64, error) {
	pgInterval := parseInterval(req.Interval)
	if pgInterval == "" {
		return r.listMetricsRaw(ctx, nodeID, column, from, to, req)
	}

	aggFunc := req.Aggregation
	if aggFunc != "avg" && aggFunc != "max" && aggFunc != "min" && aggFunc != "sum" && aggFunc != "count" {
		return nil, 0, fmt.Errorf("unsupported aggregation function: %s", aggFunc)
	}

	query := fmt.Sprintf(
		"date_trunc('second', date_trunc('minute', created_at) + "+
			"ceil(extract(epoch from created_at) / extract(epoch from interval '%s')) * "+
			"interval '%s' - date_trunc('minute', created_at)) as t, "+
			"%s(%s) as v",
		pgInterval, pgInterval, aggFunc, column,
	)

	var total int64
	countQuery := r.db.WithContext(ctx).Model(&model.EdgeNodeEngineMetrics{}).
		Select(fmt.Sprintf("COUNT(DISTINCT date_trunc('second', date_trunc('minute', created_at) + "+
			"ceil(extract(epoch from created_at) / extract(epoch from interval '%s')) * "+
			"interval '%s' - date_trunc('minute', created_at)))",
			pgInterval, pgInterval)).
		Where("node_id = ?", nodeID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var results []dto.MetricDataPoint
	err := r.db.WithContext(ctx).Model(&model.EdgeNodeEngineMetrics{}).
		Select(query).
		Where("node_id = ?", nodeID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Group("t").
		Order("t ASC").
		Scopes(
			scopes.Paginate(req.GetPage(), req.GetPageSize()),
		).
		Find(&results).Error
	if err != nil {
		return nil, 0, err
	}

	for i := range results {
		if t, ok := parseTimestamp(results[i].T); ok {
			results[i].T = t.Format(time.RFC3339)
		}
	}

	return results, total, nil
}

// DeleteOlderThan deletes all metrics records older than the given time physically.
func (r *EdgeNodeEngineMetricsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Unscoped().
		Where("created_at < ?", cutoff).
		Delete(&model.EdgeNodeEngineMetrics{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// CountByNode counts metrics records for a node within a time range.
func (r *EdgeNodeEngineMetricsRepository) CountByNode(ctx context.Context, nodeID string, from, to time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.EdgeNodeEngineMetrics{}).
		Where("node_id = ?", nodeID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Count(&total).Error
	return total, err
}

// DB returns the underlying GORM DB instance for ad-hoc queries.
func (r *EdgeNodeEngineMetricsRepository) DB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}
