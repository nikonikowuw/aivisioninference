package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

var validMetrics = map[string]struct{}{
	"cpu_usage": {}, "memory_usage": {}, "cpu_load_1m": {},
	"cpu_load_5m": {}, "cpu_load_15m": {}, "memory_used": {},
	"memory_total": {}, "disk_usage": {}, "net_rx_bytes": {},
	"net_tx_bytes": {}, "net_rx_speed": {}, "net_tx_speed": {},
	"uptime": {}, "process_count": {}, "thread_count": {},
	"temperature": {}, "current_load": {},
}

var metricsIntervals = map[string]string{
	"1m":  "1 minute",
	"5m":  "5 minutes",
	"15m": "15 minutes",
	"1h":  "1 hour",
	"6h":  "6 hours",
	"1d":  "1 day",
}

// EdgeNodeMetricsRepository handles database operations for EdgeNodeMetrics model.
type EdgeNodeMetricsRepository struct {
	db *gorm.DB
}

// NewEdgeNodeMetricsRepository creates a new EdgeNodeMetricsRepository.
func NewEdgeNodeMetricsRepository(db *gorm.DB) *EdgeNodeMetricsRepository {
	return &EdgeNodeMetricsRepository{db: db}
}

// Create inserts a new metrics record.
func (r *EdgeNodeMetricsRepository) Create(ctx context.Context, metrics *model.EdgeNodeMetrics) error {
	return r.db.WithContext(ctx).Create(metrics).Error
}

// MetricsQueryOpts 指标查询参数
type MetricsQueryOpts struct {
	NodeID      string
	Metric      string // 指标列名: "cpu_usage", "memory_usage", "cpu_load_1m", "disk_usage" etc.
	From        time.Time
	To          time.Time
	Aggregation string // "avg", "max", "min" (optional)
	Interval    string // 聚合窗口: "5m", "1h" (optional, requires Aggregation)
	Page        int
	PageSize    int
}

// Query 分页查询时序指标，支持时间范围、聚合和降采样
func (r *EdgeNodeMetricsRepository) Query(ctx context.Context, opts MetricsQueryOpts) ([]map[string]interface{}, int64, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 || opts.PageSize > 1000 {
		opts.PageSize = 500
	}

	if _, ok := validMetrics[opts.Metric]; !ok {
		return nil, 0, fmt.Errorf("unsupported metric: %s", opts.Metric)
	}

	// For disk_usage (JSONB), use a different query path
	if opts.Metric == "disk_usage" {
		return r.queryDiskUsage(ctx, opts)
	}

	db := r.db.WithContext(ctx)
	var total int64

	// Build base conditions
	baseQuery := db.Model(&model.EdgeNodeMetrics{}).
		Where("node_id = ?", opts.NodeID).
		Where("created_at BETWEEN ? AND ?", opts.From, opts.To)

	// Count (without aggregation)
	countQuery := baseQuery
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// If aggregation + interval specified, use time-bucketed query
	if opts.Aggregation != "" && opts.Interval != "" {
		interval, ok := metricsIntervals[opts.Interval]
		if !ok {
			return nil, 0, fmt.Errorf("unsupported metrics interval: %s", opts.Interval)
		}
		return r.queryAggregated(ctx, opts, interval)
	}

	// Raw data query
	var results []map[string]interface{}
	offset := (opts.Page - 1) * opts.PageSize
	if err := baseQuery.
		Select("created_at", opts.Metric).
		Order("created_at DESC").
		Offset(offset).
		Limit(opts.PageSize).
		Find(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// queryAggregated 时间窗口聚合查询
func (r *EdgeNodeMetricsRepository) queryAggregated(ctx context.Context, opts MetricsQueryOpts, interval string) ([]map[string]interface{}, int64, error) {
	var aggFn string
	switch opts.Aggregation {
	case "avg":
		aggFn = "AVG"
	case "max":
		aggFn = "MAX"
	case "min":
		aggFn = "MIN"
	default:
		aggFn = "AVG"
	}

	// Metric and aggregation identifiers come from fixed allowlists above. The
	// interval remains a bound value so user input never becomes SQL syntax.
	query := fmt.Sprintf(`
			SELECT
				date_bin(?::interval, created_at, TIMESTAMPTZ '1970-01-01') AS t,
				%s(%s) AS v
		FROM edge_node_metrics
		WHERE node_id = ? AND created_at BETWEEN ? AND ?
		GROUP BY t
		ORDER BY t DESC
	`, aggFn, opts.Metric)

	// Count of buckets
	var total int64
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT 1 FROM edge_node_metrics
			WHERE node_id = ? AND created_at BETWEEN ? AND ?
				GROUP BY date_bin(?::interval, created_at, TIMESTAMPTZ '1970-01-01')
			) AS buckets
	`)
	if err := r.db.WithContext(ctx).Raw(countQuery, opts.NodeID, opts.From, opts.To, interval).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// Paginate buckets
	offset := (opts.Page - 1) * opts.PageSize
	paginatedQuery := fmt.Sprintf(`SELECT * FROM (%s) AS sub ORDER BY t DESC LIMIT ? OFFSET ?`, query)
	var results []map[string]interface{}
	if err := r.db.WithContext(ctx).Raw(paginatedQuery, interval, opts.NodeID, opts.From, opts.To, opts.PageSize, offset).
		Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// queryDiskUsage 磁盘使用量查询（JSONB，直接透传最近一条）
func (r *EdgeNodeMetricsRepository) queryDiskUsage(ctx context.Context, opts MetricsQueryOpts) ([]map[string]interface{}, int64, error) {
	var results []map[string]interface{}
	var total int64

	offset := (opts.Page - 1) * opts.PageSize
	if err := r.db.WithContext(ctx).
		Model(&model.EdgeNodeMetrics{}).
		Where("node_id = ? AND created_at BETWEEN ? AND ?", opts.NodeID, opts.From, opts.To).
		Select("created_at, disk_usage").
		Order("created_at DESC").
		Offset(offset).
		Limit(opts.PageSize).
		Find(&results).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Model(&model.EdgeNodeMetrics{}).
		Where("node_id = ? AND created_at BETWEEN ? AND ?", opts.NodeID, opts.From, opts.To).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// DeleteOlderThan 删除早于指定时间的记录（数据保留清理）
func (r *EdgeNodeMetricsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Unscoped().
		Where("created_at < ?", cutoff).
		Delete(&model.EdgeNodeMetrics{})
	return result.RowsAffected, result.Error
}

// CountByStatus 按节点状态统计数量
type NodeStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// CountByStatus 返回各状态节点数量
func (r *EdgeNodeMetricsRepository) CountByStatus(ctx context.Context) ([]NodeStatusCount, int64, error) {
	var counts []NodeStatusCount
	var total int64

	edgeNodeRepo := NewEdgeNodeRepository(r.db)
	if err := edgeNodeRepo.DB(ctx).
		Model(&model.EdgeNode{}).
		Select("status, COUNT(*) AS count").
		Group("status").
		Find(&counts).Error; err != nil {
		return nil, 0, err
	}

	if err := edgeNodeRepo.DB(ctx).Model(&model.EdgeNode{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	return counts, total, nil
}
