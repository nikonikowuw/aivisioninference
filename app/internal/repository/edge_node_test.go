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

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
)

func setupEdgeNodeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:edge_node_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
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
			media_decode_capacity INTEGER NOT NULL DEFAULT 0,
			media_encode_capacity INTEGER NOT NULL DEFAULT 0,
			media_egress_capacity_bps INTEGER NOT NULL DEFAULT 0,
			media_metrics_ttl_seconds INTEGER NOT NULL DEFAULT 0,
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
		CREATE TABLE IF NOT EXISTS edge_node_algorithms (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			node_id TEXT NOT NULL,
			algo_package_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			install_path TEXT,
			runtime_status TEXT NOT NULL DEFAULT 'unknown',
			supports_embedding INTEGER DEFAULT 0,
			supports_face_library INTEGER DEFAULT 0,
			embedding_capacity INTEGER DEFAULT 0,
			deployed_at DATETIME,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			last_retry_at DATETIME
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_node_algo ON edge_node_algorithms(node_id, algo_package_id);
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
			domain TEXT NOT NULL,
			result_schema TEXT NOT NULL,
			capabilities_image TEXT,
			capabilities_data TEXT,
			hardware TEXT,
			description TEXT,
			package_path TEXT NOT NULL,
			extract_path TEXT NOT NULL,
			package_size INTEGER,
			package_md5 TEXT,
			so_path TEXT NOT NULL,
			a_iparams_schema TEXT,
			self_check_status TEXT NOT NULL DEFAULT 'pending',
			self_check_result TEXT,
			self_check_at DATETIME,
			status TEXT NOT NULL DEFAULT 'draft',
			ref_count INTEGER DEFAULT 0,
			is_current INTEGER DEFAULT 0,
			remark TEXT,
			cpu_usage REAL DEFAULT 0,
			memory_usage REAL DEFAULT 0
		);
	`).Error)

	return db
}

func TestEdgeNodeRepository_CRUD(t *testing.T) {
	db := setupEdgeNodeTestDB(t)
	repo := NewEdgeNodeRepository(db)
	ctx := context.Background()

	// 1. Create
	node := &model.EdgeNode{
		BaseModel:   model.BaseModel{ID: "node-1"},
		Name:        "Test Node 1",
		Endpoint:    "http://192.168.1.10:8080",
		AuthToken:   "token-1",
		Status:      model.NodeStatusOffline,
		HALPlatform: "macos",
		MaxLoad:     4,
		Enabled:     true,
	}
	err := repo.Create(ctx, node)
	require.NoError(t, err)

	// 2. FindByID
	found, err := repo.FindByID(ctx, "node-1")
	require.NoError(t, err)
	assert.Equal(t, "Test Node 1", found.Name)
	assert.Equal(t, model.NodeStatusOffline, found.Status)

	// 3. Update
	found.Status = model.NodeStatusOnline
	err = repo.Update(ctx, found)
	require.NoError(t, err)

	found2, err := repo.FindByID(ctx, "node-1")
	require.NoError(t, err)
	assert.Equal(t, model.NodeStatusOnline, found2.Status)

	// 4. UpdateHeartbeat
	now := time.Now()
	err = repo.UpdateHeartbeatFields(ctx, "node-1", map[string]interface{}{
		"status":         model.NodeStatusError,
		"last_heartbeat": now,
	})
	require.NoError(t, err)

	found3, err := repo.FindByID(ctx, "node-1")
	require.NoError(t, err)
	assert.Equal(t, model.NodeStatusError, found3.Status)
	assert.NotNil(t, found3.LastHeartbeat)

	// 5. List
	req := dto.EdgeNodeListRequest{
		Status: "error",
	}
	req.Page = 1
	req.PageSize = 10
	list, total, err := repo.List(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)

	// 6. Delete (Soft Delete)
	err = repo.Delete(ctx, "node-1")
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, "node-1")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Check if soft deleted record exists with status=disabled
	var deletedNode model.EdgeNode
	err = db.Unscoped().Where("id = ?", "node-1").First(&deletedNode).Error
	require.NoError(t, err)
	assert.Equal(t, model.NodeStatusDisabled, deletedNode.Status)
	assert.NotNil(t, deletedNode.DeletedAt)
}

func TestEdgeNodeAlgorithmRepository_Operations(t *testing.T) {
	db := setupEdgeNodeTestDB(t)
	nodeRepo := NewEdgeNodeRepository(db)
	algoRepo := NewEdgeNodeAlgorithmRepository(db)
	ctx := context.Background()

	// Seed node and algorithm package
	node := &model.EdgeNode{
		BaseModel: model.BaseModel{ID: "node-2"},
		Name:      "Node 2",
		Endpoint:  "http://127.0.0.1:8080",
		Status:    "online",
		Enabled:   true,
	}
	require.NoError(t, nodeRepo.Create(ctx, node))

	pkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "algo-pkg-1"},
		AlgorithmName: "yolov8",
		Version:       "1.0.0",
		PackageMD5:    "md5-hash",
		PackagePath:   "yolov8_v1.tar.gz",
		ExtractPath:   "/opt/algo/yolov8",
		SoPath:        "/opt/algo/yolov8/yolov8.so",
	}
	require.NoError(t, db.Create(pkg).Error)

	// 1. Create
	deployment := &model.EdgeNodeAlgorithm{
		BaseModel:     model.BaseModel{ID: "deploy-1"},
		NodeID:        "node-2",
		AlgoPackageID: "algo-pkg-1",
		Status:        model.AlgoDeployPending,
	}
	err := algoRepo.Create(ctx, deployment)
	require.NoError(t, err)

	// 2. FindByNodeAndAlgo
	found, err := algoRepo.FindByNodeAndAlgo(ctx, "node-2", "algo-pkg-1")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployPending, found.Status)
	assert.Equal(t, "yolov8", found.AlgoPackage.AlgorithmName)

	// 3. ListPendingByNode
	pendings, err := algoRepo.ListPendingByNode(ctx, "node-2")
	require.NoError(t, err)
	assert.Len(t, pendings, 1)
	assert.Equal(t, "deploy-1", pendings[0].ID)

	// 4. UpdateStatus
	err = algoRepo.UpdateStatus(ctx, "node-2", "algo-pkg-1", model.AlgoDeployDownloading)
	require.NoError(t, err)

	found2, err := algoRepo.FindByNodeAndAlgo(ctx, "node-2", "algo-pkg-1")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployDownloading, found2.Status)

	// 5. UpdateInstallInfo
	now := time.Now()
	err = algoRepo.UpdateInstallInfo(ctx, "node-2", "algo-pkg-1", "/opt/yolov8", now)
	require.NoError(t, err)

	found3, err := algoRepo.FindByNodeAndAlgo(ctx, "node-2", "algo-pkg-1")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployInstalled, found3.Status)
	assert.Equal(t, "/opt/yolov8", found3.InstallPath)
	assert.NotNil(t, found3.DeployedAt)

	// 6. FindOnlineNodesWithAlgorithm
	onlineNodes, err := nodeRepo.FindOnlineNodesWithAlgorithm(ctx, "algo-pkg-1")
	require.NoError(t, err)
	assert.Len(t, onlineNodes, 1)
	assert.Equal(t, "node-2", onlineNodes[0].ID)

	// 7. SyncInstalled
	// Update deployment status to failed first
	err = algoRepo.UpdateStatus(ctx, "node-2", "algo-pkg-1", model.AlgoDeployFailed)
	require.NoError(t, err)

	installedReport := []dto.InstalledAlgorithmInfo{
		{
			AlgoPackageID: "algo-pkg-1",
			Version:       "1.0.0",
			InstallPath:   "/opt/yolov8-synced",
			Status:        "installed",
		},
	}
	err = algoRepo.SyncInstalled(ctx, "node-2", installedReport)
	require.NoError(t, err)

	found4, err := algoRepo.FindByNodeAndAlgo(ctx, "node-2", "algo-pkg-1")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployInstalled, found4.Status)
	assert.Equal(t, "/opt/yolov8-synced", found4.InstallPath)

	// 8. FindFailedWithRetries and UpdateForRetry
	err = algoRepo.UpdateStatus(ctx, "node-2", "algo-pkg-1", model.AlgoDeployFailed)
	require.NoError(t, err)

	failedList, err := algoRepo.FindFailedWithRetries(ctx)
	require.NoError(t, err)
	assert.Len(t, failedList, 1)

	err = algoRepo.UpdateForRetry(ctx, "node-2", "algo-pkg-1", map[string]interface{}{
		"status":      model.AlgoDeployPending,
		"retry_count": 1,
	})
	require.NoError(t, err)

	found5, err := algoRepo.FindByNodeAndAlgo(ctx, "node-2", "algo-pkg-1")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployPending, found5.Status)
	assert.Equal(t, 1, found5.RetryCount)

	// 9. ListByNode
	allAlgos, err := algoRepo.ListByNode(ctx, "node-2")
	require.NoError(t, err)
	assert.Len(t, allAlgos, 1)
	assert.Equal(t, "deploy-1", allAlgos[0].ID)

	// 10. SyncInstalled failed case
	failedReport := []dto.InstalledAlgorithmInfo{
		{
			AlgoPackageID: "algo-pkg-1",
			Version:       "1.0.0",
			InstallPath:   "",
			Status:        "failed",
		},
	}
	err = algoRepo.SyncInstalled(ctx, "node-2", failedReport)
	require.NoError(t, err)

	found6, err := algoRepo.FindByNodeAndAlgo(ctx, "node-2", "algo-pkg-1")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployFailed, found6.Status)
	assert.Equal(t, "引擎端安装失败", found6.ErrorMessage)
}

func TestEdgeNodeRepository_FindOnlineNodesWithAlgorithm_LoadBalancing(t *testing.T) {
	db := setupEdgeNodeTestDB(t)
	nodeRepo := NewEdgeNodeRepository(db)
	algoRepo := NewEdgeNodeAlgorithmRepository(db)
	ctx := context.Background()

	// Create algorithm package
	pkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "algo-lb-1"},
		AlgorithmName: "yolov8",
		Version:       "1.0.0",
		PackageMD5:    "md5-lb",
		PackagePath:   "yolov8.tar.gz",
		ExtractPath:   "/opt/algo/yolov8",
		SoPath:        "/opt/algo/yolov8/yolov8.so",
	}
	require.NoError(t, db.Create(pkg).Error)

	// Create 3 nodes with different load configurations:
	// - Node A: load 1/4 (25%) - should be recommended
	// - Node B: load 3/4 (75%)
	// - Node C: load 2/4 (50%)
	// - Node D: online but no algorithm installed - should NOT be returned
	// - Node E: has algorithm but disabled - should NOT be returned
	// - Node F: has algorithm but offline - should NOT be returned

	nodes := []struct {
		id       string
		name     string
		status   string
		enabled  bool
		maxLoad  int
		currLoad int
	}{
		{"node-lb-a", "Low Load Node", "online", true, 4, 1},
		{"node-lb-b", "High Load Node", "online", true, 4, 3},
		{"node-lb-c", "Medium Load Node", "online", true, 4, 2},
		{"node-lb-d", "No Algo Node", "online", true, 4, 0},
		{"node-lb-e", "Disabled Node", "disabled", false, 4, 2},
		{"node-lb-f", "Offline Node", "offline", true, 4, 1},
	}

	for _, nd := range nodes {
		require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
			BaseModel:   model.BaseModel{ID: nd.id},
			Name:        nd.name,
			Endpoint:    "http://127.0.0.1:8080",
			Status:      nd.status,
			Enabled:     nd.enabled,
			MaxLoad:     nd.maxLoad,
			CurrentLoad: nd.currLoad,
		}))
	}

	// Install algorithm on nodes A, B, C, E, F (not D)
	for _, id := range []string{"node-lb-a", "node-lb-b", "node-lb-c", "node-lb-e", "node-lb-f"} {
		require.NoError(t, algoRepo.Create(ctx, &model.EdgeNodeAlgorithm{
			BaseModel:     model.BaseModel{ID: id + "-deploy"},
			NodeID:        id,
			AlgoPackageID: "algo-lb-1",
			Status:        model.AlgoDeployInstalled,
			InstallPath:   "/opt/algo/yolov8",
		}))
	}

	// Query online nodes with installed algorithm
	onlineNodes, err := nodeRepo.FindOnlineNodesWithAlgorithm(ctx, "algo-lb-1")
	require.NoError(t, err)

	// Should only find node A, B, C (online + enabled + installed)
	// Node D: no algorithm, Node E: disabled, Node F: offline
	assert.Len(t, onlineNodes, 3, "should find 3 online+enabled nodes with the algorithm")

	// Collect found node IDs for easier assertion
	foundIDs := make(map[string]bool)
	for _, n := range onlineNodes {
		foundIDs[n.ID] = true
	}
	assert.True(t, foundIDs["node-lb-a"], "node A should be included")
	assert.True(t, foundIDs["node-lb-b"], "node B should be included")
	assert.True(t, foundIDs["node-lb-c"], "node C should be included")
	assert.False(t, foundIDs["node-lb-d"], "node D (no algo) should NOT be included")
	assert.False(t, foundIDs["node-lb-e"], "node E (disabled) should NOT be included")
	assert.False(t, foundIDs["node-lb-f"], "node F (offline) should NOT be included")

	// Verify node A has the lowest load rate (1/4 = 25%)
	// This simulates what EdgeNodeService.RecommendNode does
	for _, n := range onlineNodes {
		if n.ID == "node-lb-a" {
			assert.Equal(t, 1, n.CurrentLoad, "node A should have current_load=1")
			assert.Equal(t, 4, n.MaxLoad, "node A should have max_load=4")
		}
		if n.ID == "node-lb-b" {
			assert.Equal(t, 3, n.CurrentLoad, "node B should have current_load=3")
		}
		if n.ID == "node-lb-c" {
			assert.Equal(t, 2, n.CurrentLoad, "node C should have current_load=2")
		}
	}
}

func TestEdgeNodeAlgorithmRepository_SyncInstalledBatch(t *testing.T) {
	db := setupEdgeNodeTestDB(t)
	algoRepo := NewEdgeNodeAlgorithmRepository(db)
	ctx := context.Background()

	require.NoError(t, algoRepo.Create(ctx, &model.EdgeNodeAlgorithm{
		BaseModel:           model.BaseModel{ID: "deploy-batch-1"},
		NodeID:              "node-batch",
		AlgoPackageID:       "algo-batch-installed",
		Status:              model.AlgoDeployDownloading,
		RuntimeStatus:       model.AlgoRuntimeUnknown,
		InstallPath:         "",
		ErrorMessage:        "old error",
		RetryCount:          1,
		EmbeddingCapacity:   1,
		SupportsEmbedding:   false,
		SupportsFaceLibrary: false,
	}))
	require.NoError(t, algoRepo.Create(ctx, &model.EdgeNodeAlgorithm{
		BaseModel:           model.BaseModel{ID: "deploy-batch-2"},
		NodeID:              "node-batch",
		AlgoPackageID:       "algo-batch-failed",
		Status:              model.AlgoDeployDownloading,
		RuntimeStatus:       model.AlgoRuntimeWarming,
		InstallPath:         "/old/path",
		RetryCount:          2,
		EmbeddingCapacity:   2,
		SupportsEmbedding:   true,
		SupportsFaceLibrary: true,
	}))

	err := algoRepo.SyncInstalled(ctx, "node-batch", []dto.InstalledAlgorithmInfo{
		{
			AlgoPackageID:       "algo-batch-installed",
			Version:             "1.0.0",
			InstallPath:         "/opt/algo/installed",
			Status:              "installed",
			RuntimeStatus:       model.AlgoRuntimeReady,
			SupportsEmbedding:   true,
			SupportsFaceLibrary: true,
			EmbeddingCapacity:   4,
		},
		{
			AlgoPackageID:       "algo-batch-failed",
			Version:             "1.0.0",
			Status:              "failed",
			SupportsEmbedding:   false,
			SupportsFaceLibrary: false,
		},
		{
			AlgoPackageID: "algo-batch-missing",
			Version:       "1.0.0",
			InstallPath:   "/opt/algo/missing",
			Status:        "installed",
		},
	})
	require.NoError(t, err)

	installed, err := algoRepo.FindByNodeAndAlgo(ctx, "node-batch", "algo-batch-installed")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployInstalled, installed.Status)
	assert.Equal(t, model.AlgoRuntimeReady, installed.RuntimeStatus)
	assert.Equal(t, "/opt/algo/installed", installed.InstallPath)
	assert.True(t, installed.SupportsEmbedding)
	assert.True(t, installed.SupportsFaceLibrary)
	assert.Equal(t, 4, installed.EmbeddingCapacity)
	assert.Empty(t, installed.ErrorMessage)
	assert.NotNil(t, installed.DeployedAt)

	failed, err := algoRepo.FindByNodeAndAlgo(ctx, "node-batch", "algo-batch-failed")
	require.NoError(t, err)
	assert.Equal(t, model.AlgoDeployFailed, failed.Status)
	assert.Equal(t, model.AlgoRuntimeFailed, failed.RuntimeStatus)
	assert.Equal(t, "引擎端安装失败", failed.ErrorMessage)
	assert.Equal(t, 2, failed.EmbeddingCapacity)
	assert.False(t, failed.SupportsEmbedding)
	assert.False(t, failed.SupportsFaceLibrary)

	_, err = algoRepo.FindByNodeAndAlgo(ctx, "node-batch", "algo-batch-missing")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestEdgeNodeRepository_FindOnlineNodesWithAlgorithm_EmptyResults(t *testing.T) {
	db := setupEdgeNodeTestDB(t)
	nodeRepo := NewEdgeNodeRepository(db)
	ctx := context.Background()

	// No nodes exist
	nodes, err := nodeRepo.FindOnlineNodesWithAlgorithm(ctx, "nonexistent-algo")
	require.NoError(t, err)
	assert.Empty(t, nodes, "should return empty slice for nonexistent algo")

	// Create a node but without algorithm installed
	require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
		BaseModel: model.BaseModel{ID: "node-empty-1"},
		Name:      "Empty Node",
		Endpoint:  "http://127.0.0.1:8080",
		Status:    "online",
		Enabled:   true,
	}))

	nodes, err = nodeRepo.FindOnlineNodesWithAlgorithm(ctx, "some-algo")
	require.NoError(t, err)
	assert.Empty(t, nodes, "should return empty slice when no node has the algorithm installed")
}

func TestEdgeNodeRepository_FindReadyEmbeddingNodesByAlgorithm(t *testing.T) {
	db := setupEdgeNodeTestDB(t)
	nodeRepo := NewEdgeNodeRepository(db)
	algoRepo := NewEdgeNodeAlgorithmRepository(db)
	ctx := context.Background()

	// Create algorithm package
	pkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "algo-emb-1"},
		AlgorithmName: "face_embedding",
		Version:       "1.0.0",
		PackageMD5:    "md5-emb",
		PackagePath:   "face_embedding.tar.gz",
		ExtractPath:   "/opt/algo/face_embedding",
		SoPath:        "/opt/algo/face_embedding/face_embedding.so",
	}
	require.NoError(t, db.Create(pkg).Error)

	// Create nodes:
	// - Node A: online, runtime=ready, supports_embedding=true  (should be found)
	// - Node B: online, runtime=installed, supports_embedding=true  (not ready → excluded)
	// - Node C: online, runtime=ready, supports_embedding=false (no embedding cap → excluded)
	// - Node D: disabled, runtime=ready, supports_embedding=true (disabled → excluded)
	require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
		BaseModel:         model.BaseModel{ID: "node-emb-a"},
		Name:              "Ready Embedding Node",
		Endpoint:          "http://127.0.0.1:8080",
		Status:            model.NodeStatusOnline,
		Enabled:           true,
		EmbeddingCapacity: 2,
	}))
	require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
		BaseModel: model.BaseModel{ID: "node-emb-b"},
		Name:      "Installed But Not Ready Node",
		Endpoint:  "http://127.0.0.1:8081",
		Status:    model.NodeStatusOnline,
		Enabled:   true,
	}))
	require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
		BaseModel: model.BaseModel{ID: "node-emb-c"},
		Name:      "Ready But No Embedding Cap",
		Endpoint:  "http://127.0.0.1:8082",
		Status:    model.NodeStatusOnline,
		Enabled:   true,
	}))
	require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
		BaseModel: model.BaseModel{ID: "node-emb-d"},
		Name:      "Disabled Embedding Node",
		Endpoint:  "http://127.0.0.1:8083",
		Status:    model.NodeStatusOnline,
		Enabled:   true,
	}))
	// Use explicit UPDATE because GORM skips the zero-value bool field on INSERT
	require.NoError(t, db.Model(&model.EdgeNode{}).Where("id = ?", "node-emb-d").Update("enabled", false).Error)

	// Install algorithm on all four nodes
	for _, id := range []string{"node-emb-a", "node-emb-b", "node-emb-c", "node-emb-d"} {
		require.NoError(t, algoRepo.Create(ctx, &model.EdgeNodeAlgorithm{
			BaseModel:     model.BaseModel{ID: id + "-deploy"},
			NodeID:        id,
			AlgoPackageID: "algo-emb-1",
			Status:        model.AlgoDeployInstalled,
		}))
	}
	// Set runtime status and capabilities
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-emb-a").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeReady, "supports_embedding": true}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-emb-b").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeInstalled, "supports_embedding": true}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-emb-c").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeReady, "supports_embedding": false}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-emb-d").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeReady, "supports_embedding": true}).Error)

	// FindReadyEmbeddingNodesByAlgorithm should return only node A
	nodes, err := nodeRepo.FindReadyEmbeddingNodesByAlgorithm(ctx, "algo-emb-1")
	require.NoError(t, err)
	require.Len(t, nodes, 1, "only node A should be a ready embedding node")
	assert.Equal(t, "node-emb-a", nodes[0].ID)
}

func TestEdgeNodeRepository_FindInstalledEmbeddingNodesByAlgorithm(t *testing.T) {
	db := setupEdgeNodeTestDB(t)
	nodeRepo := NewEdgeNodeRepository(db)
	algoRepo := NewEdgeNodeAlgorithmRepository(db)
	ctx := context.Background()

	pkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "algo-emb-2"},
		AlgorithmName: "face_embedding",
		Version:       "1.0.0",
		PackageMD5:    "md5-emb2",
		PackagePath:   "face_embedding.tar.gz",
		ExtractPath:   "/opt/algo/face_embedding",
		SoPath:        "/opt/algo/face_embedding/face_embedding.so",
	}
	require.NoError(t, db.Create(pkg).Error)

	// Create nodes with different runtime_status values
	// - Node A: runtime=installed, supports_embedding=true (should be found)
	// - Node B: runtime=ready, supports_embedding=true (excluded -- already ready)
	// - Node C: runtime=failed, supports_embedding=true (should be found)
	// - Node D: runtime=unknown, supports_embedding=true (should be found)
	// - Node E: runtime=installed, supports_embedding=false (excluded)
	for _, id := range []string{"node-inst-a", "node-inst-b", "node-inst-c", "node-inst-d", "node-inst-e"} {
		require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
			BaseModel: model.BaseModel{ID: id},
			Name:      id,
			Endpoint:  "http://127.0.0.1:8080",
			Status:    model.NodeStatusOnline,
			Enabled:   true,
		}))
		require.NoError(t, algoRepo.Create(ctx, &model.EdgeNodeAlgorithm{
			BaseModel:     model.BaseModel{ID: id + "-deploy"},
			NodeID:        id,
			AlgoPackageID: "algo-emb-2",
			Status:        model.AlgoDeployInstalled,
		}))
	}
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-inst-a").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeInstalled, "supports_embedding": true}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-inst-b").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeReady, "supports_embedding": true}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-inst-c").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeFailed, "supports_embedding": true}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-inst-d").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeUnknown, "supports_embedding": true}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-inst-e").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeInstalled, "supports_embedding": false}).Error)

	nodes, err := nodeRepo.FindInstalledEmbeddingNodesByAlgorithm(ctx, "algo-emb-2")
	require.NoError(t, err)
	// Should find nodes A, C, D (installed, failed, unknown with supports_embedding=true)
	// Node B is ready (excluded), Node E has supports_embedding=false (excluded)
	foundIDs := make(map[string]bool)
	for _, n := range nodes {
		foundIDs[n.ID] = true
	}
	assert.True(t, foundIDs["node-inst-a"], "node with runtime=installed should be included")
	assert.False(t, foundIDs["node-inst-b"], "node with runtime=ready should be excluded")
	assert.True(t, foundIDs["node-inst-c"], "node with runtime=failed should be included")
	assert.True(t, foundIDs["node-inst-d"], "node with runtime=unknown should be included")
	assert.False(t, foundIDs["node-inst-e"], "node with supports_embedding=false should be excluded")
}

func TestEdgeNodeRepository_FindReadyFaceLibraryNodesByAlgorithm(t *testing.T) {
	db := setupEdgeNodeTestDB(t)
	nodeRepo := NewEdgeNodeRepository(db)
	algoRepo := NewEdgeNodeAlgorithmRepository(db)
	ctx := context.Background()

	pkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "algo-fl-1"},
		AlgorithmName: "face_recognition",
		Version:       "1.0.0",
		PackageMD5:    "md5-fl",
		PackagePath:   "face_recognition.tar.gz",
		ExtractPath:   "/opt/algo/face_recognition",
		SoPath:        "/opt/algo/face_recognition/face_recognition.so",
	}
	require.NoError(t, db.Create(pkg).Error)

	// - Node A: runtime=ready, supports_face_library=true (should be found)
	// - Node B: runtime=ready, supports_face_library=false (excluded)
	// - Node C: runtime=installed, supports_face_library=true (not ready → excluded)
	for _, id := range []string{"node-fl-a", "node-fl-b", "node-fl-c"} {
		require.NoError(t, nodeRepo.Create(ctx, &model.EdgeNode{
			BaseModel: model.BaseModel{ID: id},
			Name:      id,
			Endpoint:  "http://127.0.0.1:8080",
			Status:    model.NodeStatusOnline,
			Enabled:   true,
		}))
		require.NoError(t, algoRepo.Create(ctx, &model.EdgeNodeAlgorithm{
			BaseModel:     model.BaseModel{ID: id + "-deploy"},
			NodeID:        id,
			AlgoPackageID: "algo-fl-1",
			Status:        model.AlgoDeployInstalled,
		}))
	}
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-fl-a").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeReady, "supports_face_library": true}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-fl-b").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeReady, "supports_face_library": false}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("node_id = ?", "node-fl-c").
		Updates(map[string]interface{}{"runtime_status": model.AlgoRuntimeInstalled, "supports_face_library": true}).Error)

	nodes, err := nodeRepo.FindReadyFaceLibraryNodesByAlgorithm(ctx, "algo-fl-1")
	require.NoError(t, err)
	require.Len(t, nodes, 1, "only node A should be a ready face library target")
	assert.Equal(t, "node-fl-a", nodes[0].ID)
}
