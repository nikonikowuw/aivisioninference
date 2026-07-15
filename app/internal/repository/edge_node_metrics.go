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

// EdgeNodeMetricsRepository handles database operations for EdgeNodeMetrics model.
type EdgeNodeMetricsRepository struct {
	db *gorm.DB
}

// NewEdgeNodeMetricsRepository creates a new EdgeNodeMetricsRepository.
func NewEdgeNodeMetricsRepository(db *gorm.DB) *EdgeNodeMetricsRepository {
	return &EdgeNodeMetricsRepository{db: db}
}

// Create inserts a new metrics record.
func (r *EdgeNodeMetricsRepository) Create(ctx context.Context, item *model.EdgeNodeMetrics) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// metricColumnMap maps metric type names to their database column names
var metricColumnMap = map[string]string{
	"cpu_usage":           "cpu_usage",
	"memory_usage":        "memory_usage",
	"disk_usage":          "disk_usage",
	"net_rx_bytes":        "net_rx_bytes",
	"net_tx_bytes":        "net_tx_bytes",
	"net_rx_speed":        "net_rx_speed",
	"net_tx_speed":        "net_tx_speed",
	"temperature":         "temperature",
	"process_count":       "process_count",
	"thread_count":        "thread_count",
	"uptime":              "uptime",
	"cpu_load_1m":         "cpu_load_1m",
	"cpu_load_5m":         "cpu_load_5m",
	"cpu_load_15m":        "cpu_load_15m",
	"worker_count":        "worker_count",
	"idle_worker_count":   "idle_worker_count",
	"active_stream_count": "active_stream_count",
	"current_load":        "current_load",
}

// ListMetrics returns paginated time-series metrics for a specific node and metric type.
func (r *EdgeNodeMetricsRepository) ListMetrics(ctx context.Context, nodeID string, req dto.MetricQueryRequest) ([]dto.MetricDataPoint, int64, error) {
	from, err := req.GetFromTime()
	if err != nil {
		return nil, 0, fmt.Errorf("invalid from time: %w", err)
	}
	to, err := req.GetToTime()
	if err != nil {
		return nil, 0, fmt.Errorf("invalid to time: %w", err)
	}

	column, ok := metricColumnMap[req.Metric]
	if !ok {
		return nil, 0, fmt.Errorf("unsupported metric type: %s", req.Metric)
	}

	// If aggregation is requested, use a bucketed query
	if req.Aggregation != "" && req.Interval != "" {
		return r.listMetricsAggregated(ctx, nodeID, column, from, to, req)
	}

	return r.listMetricsRaw(ctx, nodeID, column, from, to, req)
}

// listMetricsRaw returns raw (non-aggregated) data points
func (r *EdgeNodeMetricsRepository) listMetricsRaw(ctx context.Context, nodeID string, column string, from, to time.Time, req dto.MetricQueryRequest) ([]dto.MetricDataPoint, int64, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).Model(&model.EdgeNodeMetrics{}).
		Where("node_id = ?", nodeID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to)

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf("created_at as t, %s as v", column)

	var results []dto.MetricDataPoint
	err := r.db.WithContext(ctx).Model(&model.EdgeNodeMetrics{}).
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

	// Format timestamps to ISO8601
	for i := range results {
		if t, ok := parseTimestamp(results[i].T); ok {
			results[i].T = t.Format(time.RFC3339)
		}
	}

	return results, total, nil
}

// listMetricsAggregated returns aggregated data points (avg/max/min by time window)
func (r *EdgeNodeMetricsRepository) listMetricsAggregated(ctx context.Context, nodeID string, column string, from, to time.Time, req dto.MetricQueryRequest) ([]dto.MetricDataPoint, int64, error) {
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
	countQuery := r.db.WithContext(ctx).Model(&model.EdgeNodeMetrics{}).
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
	err := r.db.WithContext(ctx).Model(&model.EdgeNodeMetrics{}).
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

// DeleteOlderThan deletes all metrics records older than the given time.
func (r *EdgeNodeMetricsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Unscoped().
		Where("created_at < ?", cutoff).
		Delete(&model.EdgeNodeMetrics{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// CountByNode counts metrics records for a node within a time range.
func (r *EdgeNodeMetricsRepository) CountByNode(ctx context.Context, nodeID string, from, to time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.EdgeNodeMetrics{}).
		Where("node_id = ?", nodeID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Count(&total).Error
	return total, err
}

// CountViolationsInWindow counts metrics records where a threshold condition
// is violated within a given time window. Used by the alert engine for duration checks.
func (r *EdgeNodeMetricsRepository) CountViolationsInWindow(ctx context.Context, nodeID string, column string, operator string, threshold float64, from, to time.Time) (int64, error) {
	condition := fmt.Sprintf("%s %s ?", column, operator)
	var count int64
	err := r.db.WithContext(ctx).Model(&model.EdgeNodeMetrics{}).
		Where("node_id = ?", nodeID).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Where(condition, threshold).
		Count(&count).Error
	return count, err
}

// DB returns the underlying GORM DB instance for ad-hoc queries.
func (r *EdgeNodeMetricsRepository) DB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// GetLatestHostMetricsByNodes returns a map of the latest EdgeNodeMetrics for each provided nodeID in a single query.
func (r *EdgeNodeMetricsRepository) GetLatestHostMetricsByNodes(ctx context.Context, nodeIDs []string) (map[string]*model.EdgeNodeMetrics, error) {
	result := make(map[string]*model.EdgeNodeMetrics, len(nodeIDs))
	if len(nodeIDs) == 0 {
		return result, nil
	}
	var metrics []model.EdgeNodeMetrics
	err := r.db.WithContext(ctx).
		Where("(node_id, created_at) IN (SELECT node_id, MAX(created_at) FROM edge_node_metrics WHERE node_id IN (?) GROUP BY node_id)", nodeIDs).
		Find(&metrics).Error
	if err != nil {
		// Fallback if driver does not support tuple IN subquery
		for _, id := range nodeIDs {
			var m model.EdgeNodeMetrics
			if e := r.db.WithContext(ctx).Where("node_id = ?", id).Order("created_at DESC").First(&m).Error; e == nil {
				result[id] = &m
			}
		}
		return result, nil
	}
	for i := range metrics {
		result[metrics[i].NodeID] = &metrics[i]
	}
	return result, nil
}

// CountByStatus returns count of nodes by status (overview stats)
type NodeStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// CountByStatus returns the count of edge nodes grouped by status.
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

// parseTimestamp attempts to parse a timestamp string
func parseTimestamp(s string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseInterval converts a human-readable interval like "5m" or "1h" to PostgreSQL interval
func parseInterval(interval string) string {
	if len(interval) < 2 {
		return ""
	}
	unit := interval[len(interval)-1]
	val := interval[:len(interval)-1]
	switch unit {
	case 'm':
		return val + " minutes"
	case 'h':
		return val + " hours"
	case 'd':
		return val + " days"
	default:
		return ""
	}
}
