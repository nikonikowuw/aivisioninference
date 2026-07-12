package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
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
	base := MetricsQueryOpts{
		NodeID: "node-1", From: time.Now().Add(-time.Hour), To: time.Now(),
		Page: 1, PageSize: 20,
	}

	invalidMetric := base
	invalidMetric.Metric = "cpu_usage; DROP TABLE edge_node_metrics"
	_, _, err := repo.Query(ctx, invalidMetric)
	require.ErrorContains(t, err, "unsupported metric")

	invalidInterval := base
	invalidInterval.Metric = "cpu_usage"
	invalidInterval.Aggregation = "avg"
	invalidInterval.Interval = "5m'); DROP TABLE edge_node_metrics; --"
	_, _, err = repo.Query(ctx, invalidInterval)
	require.ErrorContains(t, err, "unsupported metrics interval")

	diskQuery := base
	diskQuery.Metric = "disk_usage"
	_, _, err = repo.Query(ctx, diskQuery)
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
	require.NoError(t, db.Unscoped().Model(&model.EdgeNodeMetrics{}).Count(&count).Error)
	require.Zero(t, count)
}
