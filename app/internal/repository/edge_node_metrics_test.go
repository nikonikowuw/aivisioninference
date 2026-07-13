package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
)

func setupEdgeNodeMetricsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:edge_node_metrics_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE edge_node_metrics (
			id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME,
			deleted_at DATETIME, created_by TEXT, updated_by TEXT, node_id TEXT NOT NULL,
			cpu_usage REAL DEFAULT 0, memory_usage REAL DEFAULT 0,
			disk_usage TEXT DEFAULT '[]'
		)
	`).Error)
	return db
}

func TestEdgeNodeMetricsRepositoryQueryValidation(t *testing.T) {
	repo := NewEdgeNodeMetricsRepository(setupEdgeNodeMetricsTestDB(t))
	ctx := context.Background()
	now := time.Now()
	from := now.Add(-time.Hour)
	to := now
	nodeID := "node-1"

	invalidMetric := dto.MetricQueryRequest{
		Metric:   "cpu_usage; DROP TABLE edge_node_metrics",
		From:     from.Format(time.RFC3339),
		To:       to.Format(time.RFC3339),
		PageRequest: dto.PageRequest{Page: 1, PageSize: 20},
	}
	_, _, err := repo.ListMetrics(ctx, nodeID, invalidMetric)
	require.ErrorContains(t, err, "unsupported metric type")

	// Invalid interval falls back to raw query gracefully (no SQL injection risk)
	fallbackInterval := dto.MetricQueryRequest{
		Metric:      "cpu_usage",
		From:        from.Format(time.RFC3339),
		To:          to.Format(time.RFC3339),
		Aggregation: "avg",
		Interval:    "5m'); DROP TABLE edge_node_metrics; --",
		PageRequest: dto.PageRequest{Page: 1, PageSize: 20},
	}
	_, _, err = repo.ListMetrics(ctx, nodeID, fallbackInterval)
	require.NoError(t, err)

	validMetric := dto.MetricQueryRequest{
		Metric:   "cpu_usage",
		From:     from.Format(time.RFC3339),
		To:       to.Format(time.RFC3339),
		PageRequest: dto.PageRequest{Page: 1, PageSize: 20},
	}
	_, _, err = repo.ListMetrics(ctx, nodeID, validMetric)
	require.NoError(t, err)
}

func TestEdgeNodeMetricsRepositoryDeleteOlderThanHardDeletes(t *testing.T) {
	db := setupEdgeNodeMetricsTestDB(t)
	repo := NewEdgeNodeMetricsRepository(db)
	ctx := context.Background()
	old := time.Now().Add(-8 * 24 * time.Hour)
	require.NoError(t, db.Exec(
		"INSERT INTO edge_node_metrics (id, node_id, created_at) VALUES (?, ?, ?)",
		"metric-1", "node-1", old,
	).Error)

	deleted, err := repo.DeleteOlderThan(ctx, time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	var count int64
	require.NoError(t, db.Unscoped().Table("edge_node_metrics").Count(&count).Error)
	require.Equal(t, int64(1), count)
}
