package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
	"github.com/niko-admin/niko-admin/internal/repository"
)

type fixedMediaMetrics struct {
	now time.Time
}

func (m fixedMediaMetrics) GetFreshNodeMediaMetrics(string, time.Time, time.Duration) (*controlproto.EngineMetricsSnapshot, bool) {
	return &controlproto.EngineMetricsSnapshot{PreviewCapacity: 1, PreviewCapacityValid: true, TimestampNS: uint64(m.now.UnixNano())}, true
}

func TestMediaCapacitySchedulerDoesNotOversubscribeLastSlot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "media-capacity.db")
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", path)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(8)

	require.NoError(t, db.Exec(`CREATE TABLE edge_nodes (
		id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
		created_by TEXT, updated_by TEXT, name TEXT, endpoint TEXT, auth_token TEXT,
		status TEXT, enabled INTEGER, media_decode_capacity INTEGER,
		media_encode_capacity INTEGER, media_egress_capacity_bps INTEGER,
		media_metrics_ttl_seconds INTEGER, cpu_usage REAL DEFAULT 0, memory_usage REAL DEFAULT 0
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE media_capacity_reservations (
		id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
		created_by TEXT, updated_by TEXT, stream_key TEXT NOT NULL UNIQUE, node_id TEXT NOT NULL,
		decode_slots INTEGER NOT NULL, encode_slots INTEGER NOT NULL, egress_bps INTEGER NOT NULL, preview_slots INTEGER NOT NULL DEFAULT 1,
		status TEXT NOT NULL, lease_expires DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO edge_nodes
		(id, name, endpoint, auth_token, status, enabled, media_decode_capacity, media_encode_capacity, media_egress_capacity_bps, media_metrics_ttl_seconds)
		VALUES ('node-1', 'node-1', 'http://node-1', 'token', 'online', 1, 1, 1, 100, 60)`).Error)

	now := time.Now()
	newScheduler := func() *MediaCapacityScheduler {
		return NewMediaCapacityScheduler(db, repository.NewEdgeNodeRepository(db), repository.NewMediaCapacityReservationRepository(db), fixedMediaMetrics{now: now})
	}

	var wg sync.WaitGroup
	results := make(chan *model.MediaCapacityReservation, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			reservation, _, err := newScheduler().Reserve(context.Background(), fmt.Sprintf("stream-%d", index), time.Minute)
			results <- reservation
			errs <- err
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	granted := 0
	for reservation := range results {
		if reservation != nil {
			granted++
		}
	}
	require.Equal(t, 1, granted)

	var count int64
	require.NoError(t, db.Model(&model.MediaCapacityReservation{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestMediaCapacitySchedulerReclaimsExpiredLease(t *testing.T) {
	db := setupMediaCapacitySchedulerDB(t)
	now := time.Now()
	scheduler := NewMediaCapacityScheduler(db, repository.NewEdgeNodeRepository(db), repository.NewMediaCapacityReservationRepository(db), fixedMediaMetrics{now: now})
	scheduler.now = func() time.Time { return now }

	first, _, err := scheduler.Reserve(context.Background(), "stream-reused", time.Second)
	require.NoError(t, err)
	require.NotNil(t, first)

	now = now.Add(2 * time.Second)
	second, _, err := scheduler.Reserve(context.Background(), "stream-reused", time.Second)
	require.NoError(t, err)
	require.NotNil(t, second)

	var count int64
	require.NoError(t, db.Model(&model.MediaCapacityReservation{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func setupMediaCapacitySchedulerDB(t *testing.T) *gorm.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "media-capacity-single.db")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", path)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE edge_nodes (
		id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
		created_by TEXT, updated_by TEXT, name TEXT, endpoint TEXT, auth_token TEXT,
		status TEXT, enabled INTEGER, media_decode_capacity INTEGER,
		media_encode_capacity INTEGER, media_egress_capacity_bps INTEGER,
		media_metrics_ttl_seconds INTEGER, cpu_usage REAL DEFAULT 0, memory_usage REAL DEFAULT 0
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE media_capacity_reservations (
		id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
		created_by TEXT, updated_by TEXT, stream_key TEXT NOT NULL UNIQUE, node_id TEXT NOT NULL,
		decode_slots INTEGER NOT NULL, encode_slots INTEGER NOT NULL, egress_bps INTEGER NOT NULL, preview_slots INTEGER NOT NULL DEFAULT 1,
		status TEXT NOT NULL, lease_expires DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO edge_nodes
		(id, name, endpoint, auth_token, status, enabled, media_decode_capacity, media_encode_capacity, media_egress_capacity_bps, media_metrics_ttl_seconds)
		VALUES ('node-1', 'node-1', 'http://node-1', 'token', 'online', 1, 1, 1, 100, 60)`).Error)
	return db
}
