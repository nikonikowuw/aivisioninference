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
	"github.com/niko-admin/niko-admin/internal/repository"
)

func setupRetryTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:retry_task_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
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
			engine_version TEXT,
			uptime INTEGER,
			enabled INTEGER DEFAULT 1,
			remark TEXT
		);
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS edge_node_algorithms (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			node_id TEXT NOT NULL,
			algo_package_id TEXT NOT NULL,
			status TEXT NOT NULL,
			install_path TEXT,
			runtime_status TEXT NOT NULL DEFAULT 'unknown',
			supports_embedding INTEGER DEFAULT 0,
			supports_face_library INTEGER DEFAULT 0,
			embedding_capacity INTEGER DEFAULT 0,
			error_message TEXT,
			deployed_at DATETIME,
			retry_count INTEGER DEFAULT 0,
			last_retry_at DATETIME
		);
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS algorithm_packages (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			algorithm_name TEXT NOT NULL,
			algorithm_alias TEXT,
			version TEXT NOT NULL,
			domain TEXT NOT NULL DEFAULT '',
			result_schema TEXT NOT NULL DEFAULT '',
			capabilities_image TEXT,
			capabilities_data TEXT,
			hardware TEXT,
			description TEXT,
			package_path TEXT NOT NULL,
			extract_path TEXT NOT NULL DEFAULT '',
			package_size INTEGER DEFAULT 0,
			package_md5 TEXT,
			so_path TEXT NOT NULL DEFAULT '',
			a_iparams_schema TEXT,
			self_check_status TEXT NOT NULL DEFAULT 'pending',
			self_check_result TEXT,
			self_check_at DATETIME,
			status TEXT NOT NULL DEFAULT 'draft',
			ref_count INTEGER DEFAULT 0,
			is_current INTEGER DEFAULT 0,
			remark TEXT
		);
	`).Error)

	return db
}

func TestEdgeNodeAlgorithmRetryTask_handleAlgorithmRetry(t *testing.T) {
	db := setupRetryTaskTestDB(t)
	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	algoPkgRepo := repository.NewAlgorithmPackageRepository(db)
	taskHandler := NewEdgeNodeAlgorithmRetryTask(nodeAlgoRepo, nodeRepo)

	ctx := context.Background()

	// Seed online node
	node := &model.EdgeNode{
		BaseModel: model.BaseModel{ID: "node-1"},
		Name:      "Online Node",
		Endpoint:  "http://127.0.0.1:8080",
		Status:    model.NodeStatusOnline,
	}
	require.NoError(t, nodeRepo.Create(ctx, node))

	// Seed algorithm package
	pkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "pkg-1"},
		AlgorithmName: "test-algo",
		Version:       "1.0.0",
		PackagePath:   "/path/to/pkg.tar.gz",
		PackageMD5:    "md5md5md5md5md5md5md5md5",
		Status:        "active",
	}
	require.NoError(t, algoPkgRepo.Create(ctx, pkg))

	// Seed a failed deployment that is ready to retry
	failedTime := time.Now().Add(-10 * time.Minute)
	dep := &model.EdgeNodeAlgorithm{
		BaseModel:     model.BaseModel{ID: "deploy-1", UpdatedAt: failedTime},
		NodeID:        "node-1",
		AlgoPackageID: "pkg-1",
		Status:        model.AlgoDeployFailed,
		RetryCount:    0,
		LastRetryAt:   &failedTime,
	}
	require.NoError(t, nodeAlgoRepo.Create(ctx, dep))

	// Seed a failed deployment that is NOT ready to retry (recently retried)
	recentTime := time.Now().Add(-1 * time.Minute)
	depRecent := &model.EdgeNodeAlgorithm{
		BaseModel:     model.BaseModel{ID: "deploy-recent", UpdatedAt: recentTime},
		NodeID:        "node-1",
		AlgoPackageID: "pkg-recent",
		Status:        model.AlgoDeployFailed,
		RetryCount:    0,
		LastRetryAt:   &recentTime,
	}
	require.NoError(t, nodeAlgoRepo.Create(ctx, depRecent))

	// Execute task
	asynqTask := asynq.NewTask(TypeEdgeNodeAlgorithmRetry, nil)
	err := taskHandler.handleAlgorithmRetry(ctx, asynqTask)
	require.NoError(t, err)

	// Verify
	d1, err := nodeAlgoRepo.FindByNodeAndAlgo(ctx, "node-1", "pkg-1")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployPending, d1.Status)
	assert.Equal(t, 1, d1.RetryCount)

	dRecent, err := nodeAlgoRepo.FindByNodeAndAlgo(ctx, "node-1", "pkg-recent")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployFailed, dRecent.Status)
	assert.Equal(t, 0, dRecent.RetryCount)
}
