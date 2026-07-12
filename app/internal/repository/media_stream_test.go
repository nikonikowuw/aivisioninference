package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

func setupMediaStreamTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:media_stream_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE media_streams (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			device_id TEXT NOT NULL,
			task_id TEXT,
			zlm_app TEXT NOT NULL DEFAULT 'live',
			zlm_stream TEXT NOT NULL,
			zlm_vhost TEXT DEFAULT '__defaultVhost__',
			zlm_schema TEXT DEFAULT 'rtsp',
			play_url_rtsp TEXT,
			play_url_rtmp TEXT,
			play_url_flv TEXT,
			play_url_ws_flv TEXT,
			play_url_webrtc TEXT,
			play_url_hls TEXT,
			play_url_mp4 TEXT,
			external_key TEXT,
			status TEXT DEFAULT 'inactive',
			consumer_count INTEGER DEFAULT 0,
			last_consumer_reason TEXT,
			started_at DATETIME,
			stopped_at DATETIME
		)
	`).Error)
	for _, field := range []string{"NodeID", "StreamMode", "DecodeSlots", "EncodeSlots", "EgressBps"} {
		require.NoError(t, db.Migrator().AddColumn(&model.MediaStream{}, field))
	}
	return db
}

func TestMediaStreamRepositoryAssignmentAndRecovery(t *testing.T) {
	db := setupMediaStreamTestDB(t)
	repo := NewMediaStreamRepository(db)
	ctx := context.Background()

	active := &model.MediaStream{
		BaseModel:   model.BaseModel{ID: "stream-active"},
		DeviceID:    "device-1",
		StreamMode:  model.MediaStreamModePreview,
		DecodeSlots: 1,
		EgressBps:   4_000_000,
		ZLMApp:      "live",
		ZLMStream:   "device-1",
		ZLMVhost:    "__defaultVhost__",
		Status:      "active",
	}
	require.NoError(t, repo.Create(ctx, active))
	require.NoError(t, repo.UpdateAssignment(ctx, active.ID, "node-1"))

	found, err := repo.FindActiveByDeviceAndNode(ctx, "device-1", "node-1")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, model.MediaStreamModePreview, found.StreamMode)
	assert.Equal(t, 1, found.DecodeSlots)
	assert.Equal(t, int64(4_000_000), found.EgressBps)
	require.NotNil(t, found.NodeID)
	assert.Equal(t, "node-1", *found.NodeID)

	missing, err := repo.FindActiveByDeviceAndNode(ctx, "device-1", "node-2")
	require.NoError(t, err)
	assert.Nil(t, missing)

	inactiveNode := "node-2"
	require.NoError(t, repo.Create(ctx, &model.MediaStream{
		BaseModel:  model.BaseModel{ID: "stream-inactive"},
		DeviceID:   "device-2",
		NodeID:     &inactiveNode,
		StreamMode: model.MediaStreamModeMixed,
		ZLMApp:     "live",
		ZLMStream:  "device-2",
		ZLMVhost:   "__defaultVhost__",
		Status:     "inactive",
	}))

	recoverable, err := repo.FindAssignedForRecovery(ctx)
	require.NoError(t, err)
	require.Len(t, recoverable, 1)
	assert.Equal(t, active.ID, recoverable[0].ID)
}

func TestMediaStreamMigrationAddsSchedulerFields(t *testing.T) {
	db := setupMediaStreamTestDB(t)
	migrator := db.Migrator()

	for _, field := range []string{"NodeID", "StreamMode", "DecodeSlots", "EncodeSlots", "EgressBps"} {
		assert.True(t, migrator.HasColumn(&model.MediaStream{}, field), field)
	}
}
