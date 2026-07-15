package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
)

func setupTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:task_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS edge_nodes (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			endpoint TEXT NOT NULL,
			auth_token TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'offline',
			last_heartbeat DATETIME,
			cpu_model TEXT,
			gpu_model TEXT,
			hal_platform TEXT,
			total_memory INTEGER,
			current_load INTEGER DEFAULT 0,
			max_load INTEGER DEFAULT 1,
			embedding_capacity INTEGER DEFAULT 1,
			media_decode_capacity INTEGER DEFAULT 0,
			media_encode_capacity INTEGER DEFAULT 0,
			media_egress_capacity_bps INTEGER DEFAULT 0,
			media_metrics_ttl_seconds INTEGER DEFAULT 0,
			engine_version TEXT,
			uptime INTEGER,
			enabled INTEGER DEFAULT 1,
			remark TEXT,
			runtime_error TEXT,
			ssh_port INTEGER DEFAULT 22,
			ssh_private_key TEXT,
			cpu_usage REAL DEFAULT 0,
			memory_usage REAL DEFAULT 0
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_vision_tasks (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			name TEXT,
			status TEXT,
			target_node_id TEXT,
			suspended_reason TEXT,
			error_reason TEXT
		);
	`).Error)

	return db
}

func TestEdgeNodeStatusTask_handleEdgeNodeStatusCheck(t *testing.T) {
	db := setupTaskTestDB(t)
	nodeRepo := repository.NewEdgeNodeRepository(db)
	taskRepo := repository.NewAIVisionTaskRepository(db)
	store := service.NewMemoryHeartbeatStore()
	runtimeStateStore := service.NewEdgeNodeRuntimeStateStore(cache.NewMemoryCache(0))
	taskHandler := NewEdgeNodeStatusTask(nodeRepo, taskRepo, nil, 15, store, runtimeStateStore)

	ctx := context.Background()
	now := time.Now()

	// Node 1: Online — record heartbeat 30s ago (should be expired)
	oldHB := now.Add(-30 * time.Second)
	node1 := &model.EdgeNode{
		BaseModel:     model.BaseModel{ID: "node-timed-out"},
		Name:          "Timed Out Node",
		Endpoint:      "http://127.0.0.1:8080",
		Status:        model.NodeStatusOnline,
		LastHeartbeat: &oldHB,
	}
	require.NoError(t, store.Record(ctx, node1.ID, oldHB))

	// Node 2: Online — record heartbeat 5s ago (still fresh)
	freshHB := now.Add(-5 * time.Second)
	node2 := &model.EdgeNode{
		BaseModel:     model.BaseModel{ID: "node-healthy"},
		Name:          "Healthy Node",
		Endpoint:      "http://127.0.0.1:8081",
		Status:        model.NodeStatusOnline,
		LastHeartbeat: &freshHB,
	}
	require.NoError(t, store.Record(ctx, node2.ID, freshHB))

	// Node 3: Already offline, no heartbeat record
	node3 := &model.EdgeNode{
		BaseModel: model.BaseModel{ID: "node-already-offline"},
		Name:      "Already Offline Node",
		Endpoint:  "http://127.0.0.1:8082",
		Status:    model.NodeStatusOffline,
	}

	// Node 4: stale store entry but fresh durable heartbeat (concurrent heartbeat won the race).
	node4 := &model.EdgeNode{
		BaseModel:     model.BaseModel{ID: "node-concurrent-heartbeat"},
		Name:          "Concurrent Heartbeat Node",
		Endpoint:      "http://127.0.0.1:8083",
		Status:        model.NodeStatusOnline,
		LastHeartbeat: &now,
	}
	require.NoError(t, store.Record(ctx, node4.ID, oldHB))

	require.NoError(t, nodeRepo.Create(ctx, node1))
	require.NoError(t, nodeRepo.Create(ctx, node2))
	require.NoError(t, nodeRepo.Create(ctx, node3))
	require.NoError(t, nodeRepo.Create(ctx, node4))
	runtimeStateStore.LoadFromNode(ctx, node1)
	runtimeStateStore.LoadFromNode(ctx, node4)

	// Execute Task
	asynqTask := asynq.NewTask(TypeEdgeNodeStatusCheck, nil)
	err := taskHandler.handleEdgeNodeStatusCheck(ctx, asynqTask)
	require.NoError(t, err)

	// Verify node states
	n1, err := nodeRepo.FindByID(ctx, "node-timed-out")
	require.NoError(t, err)
	assert.Equal(t, model.NodeStatusOffline, n1.Status) // Node 1 should be offline now

	n2, err := nodeRepo.FindByID(ctx, "node-healthy")
	require.NoError(t, err)
	assert.Equal(t, model.NodeStatusOnline, n2.Status) // Node 2 should remain online

	n3, err := nodeRepo.FindByID(ctx, "node-already-offline")
	require.NoError(t, err)
	assert.Equal(t, model.NodeStatusOffline, n3.Status) // Node 3 should remain offline

	n4, err := nodeRepo.FindByID(ctx, "node-concurrent-heartbeat")
	require.NoError(t, err)
	assert.Equal(t, model.NodeStatusOnline, n4.Status)

	_, cached := runtimeStateStore.Get(ctx, node1.ID)
	assert.False(t, cached, "timed-out node cache must be invalidated")
	_, cached = runtimeStateStore.Get(ctx, node4.ID)
	assert.True(t, cached, "non-transitioned node cache must remain available")

	// Verify stale heartbeat entry for node3 was cleaned up
	remaining, err := store.GetExpired(ctx, now)
	require.NoError(t, err)
	assert.NotContains(t, remaining, "node-already-offline")

	// Verify node-timed-out was removed from store after offline processing
	assert.NotContains(t, remaining, "node-timed-out")
	assert.Contains(t, remaining, "node-concurrent-heartbeat")
}

func TestHandleNodeOffline_IsConditionalAndIdempotent(t *testing.T) {
	db := setupTaskTestDB(t)
	nodeRepo := repository.NewEdgeNodeRepository(db)
	taskRepo := repository.NewAIVisionTaskRepository(db)
	ctx := context.Background()
	now := time.Now()
	node := model.EdgeNode{
		BaseModel: model.BaseModel{ID: "node-race"}, Name: "Race Node", Endpoint: "http://127.0.0.1",
		Status: model.NodeStatusOnline, LastHeartbeat: &now,
	}
	require.NoError(t, nodeRepo.Create(ctx, &node))
	require.NoError(t, db.Exec(`INSERT INTO ai_vision_tasks (id, name, status, target_node_id) VALUES (?, ?, ?, ?)`,
		"task-race", "Race Task", model.TaskStatusRunning, node.ID).Error)

	staleCutoff := now.Add(-time.Second)
	transitioned, err := service.HandleNodeOffline(ctx, nodeRepo, taskRepo, nil, service.NewMemoryHeartbeatStore(), node,
		model.SuspendedReasonNodeOffline, "offline", &staleCutoff)
	require.NoError(t, err)
	assert.False(t, transitioned)
	freshNode, err := nodeRepo.FindByID(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, model.NodeStatusOnline, freshNode.Status)

	futureCutoff := now.Add(time.Second)
	transitioned, err = service.HandleNodeOffline(ctx, nodeRepo, taskRepo, nil, service.NewMemoryHeartbeatStore(), node,
		model.SuspendedReasonNodeOffline, "offline", &futureCutoff)
	require.NoError(t, err)
	assert.True(t, transitioned)

	transitioned, err = service.HandleNodeOffline(ctx, nodeRepo, taskRepo, nil, service.NewMemoryHeartbeatStore(), node,
		model.SuspendedReasonNodeOffline, "offline", &futureCutoff)
	require.NoError(t, err)
	assert.False(t, transitioned)

	var task model.AIVisionTask
	require.NoError(t, db.First(&task, "id = ?", "task-race").Error)
	assert.Equal(t, model.TaskStatusSuspended, task.Status)
	require.NotNil(t, task.SuspendedReason)
	assert.Equal(t, model.SuspendedReasonNodeOffline, *task.SuspendedReason)
}
