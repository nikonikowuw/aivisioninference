package service

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
	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/pkg/storage"
	"gorm.io/datatypes"
)

func setupServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
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
			remark TEXT
		);
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS ai_vision_tasks (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			schedule_id TEXT,
			device_channel_id TEXT,
			algo_package_id TEXT,
			target_node_id TEXT NOT NULL,
			start_date DATE,
			end_date DATE,
			time_windows TEXT,
			a_iparams TEXT,
			roi_regions TEXT,
			mark_regions TEXT,
			line_regions TEXT,
			error_reason TEXT
		);
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS devices (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			device_name TEXT,
			access_type TEXT,
			rtsp_url TEXT,
			gb28181_device_id TEXT,
			gb28181_channel_id TEXT,
			username TEXT,
			password TEXT,
			manufacturer TEXT,
			model TEXT,
			firmware_version TEXT,
			status TEXT,
			enabled BOOLEAN,
			latitude NUMERIC,
			longitude NUMERIC,
			location_desc TEXT,
			last_online_at DATETIME,
			last_offline_at DATETIME,
			last_error_code TEXT,
			last_error_message TEXT,
			external_key TEXT,
			remark TEXT,
			version INTEGER,
			auto_infer BOOLEAN,
			parent_nvr_id TEXT
		);
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS smart_records (
			record_id TEXT PRIMARY KEY,
			record_type TEXT,
			capture_time DATETIME,
			task_id TEXT,
			task_name TEXT,
			device_id TEXT,
			device_name TEXT,
			algorithm_name TEXT,
			algorithm_version TEXT,
			category_code INTEGER,
			confidence NUMERIC,
			person_record_id TEXT,
			person_name TEXT,
			similarity NUMERIC,
			identity_id TEXT,
			triggered_line_ids TEXT,
			direction TEXT,
			alarm_type TEXT,
			alarm_level TEXT,
			alarm_status TEXT,
			alarm_major TEXT,
			correlation_id TEXT,
			business_tags TEXT,
			snapshot_image_url TEXT,
			target_crop_url TEXT,
			background_image_url TEXT,
			raw_result TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT
		);
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS device_group_members (
			device_id TEXT,
			group_id TEXT,
			created_at DATETIME
		);
	`).Error)

	return db
}

func TestEdgeNodeService_Lifecycle(t *testing.T) {
	db := setupServiceTestDB(t)

	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	pkgRepo := repository.NewAlgorithmPackageRepository(db)
	taskRepo := repository.NewAIVisionTaskRepository(db)
	jwtManager := jwt.NewManager("my-very-secure-jwt-secret-at-least-32-chars", "niko-admin", "niko-admin", 3600, 86400, nil)

	deviceRepo := repository.NewDeviceRepository(db)
	smartRecordRepo := repository.NewSmartRecordRepository(db)
	fileStorage, _ := storage.NewLocalStorage(".", "http://minio:9000/aivision-algorithms")
	svc := NewEdgeNodeService(nodeRepo, nodeAlgoRepo, pkgRepo, taskRepo, deviceRepo, smartRecordRepo, jwtManager, fileStorage, nil)
	ctx := context.Background()

	// 1. Create Node
	createReq := dto.CreateEdgeNodeRequest{
		Name:                   "Edge Node 1",
		Endpoint:               "http://192.168.1.10:8080",
		MaxLoad:                5,
		MediaDecodeCapacity:    8,
		MediaEncodeCapacity:    4,
		MediaEgressCapacityBPS: 100_000_000,
		MediaMetricsTTLSeconds: 15,
		Remark:                 "Init",
	}
	node, token, err := svc.Create(ctx, createReq)
	require.NoError(t, err)
	assert.NotEmpty(t, node.ID)
	assert.NotEmpty(t, token)
	assert.Equal(t, model.NodeStatusOffline, node.Status)
	assert.Equal(t, 8, node.MediaDecodeCapacity)
	assert.Equal(t, int64(100_000_000), node.MediaEgressCapacityBPS)

	// Create duplicate
	_, _, err = svc.Create(ctx, createReq)
	assert.Error(t, err) // duplicate name error

	// 2. GetByID
	found, err := svc.GetByID(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, "Edge Node 1", found.Name)

	_, err = svc.GetByID(ctx, "invalid-id")
	assert.Error(t, err)

	// 3. List
	listReq := dto.EdgeNodeListRequest{
		Status: "offline",
	}
	listReq.Page = 1
	listReq.PageSize = 10
	list, total, err := svc.List(ctx, listReq)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)

	// 4. Update
	enabledVal := true
	zero := 0
	zeroBPS := int64(0)
	updateReq := dto.UpdateEdgeNodeRequest{
		Name:                   "Edge Node 1 Updated",
		Enabled:                &enabledVal,
		MediaDecodeCapacity:    &zero,
		MediaEncodeCapacity:    &zero,
		MediaEgressCapacityBPS: &zeroBPS,
		MediaMetricsTTLSeconds: &zero,
		Remark:                 "Updated remark",
	}
	err = svc.Update(ctx, node.ID, updateReq)
	require.NoError(t, err)

	found2, _ := svc.GetByID(ctx, node.ID)
	assert.Equal(t, "Edge Node 1 Updated", found2.Name)
	assert.Equal(t, "Updated remark", found2.Remark)
	assert.Zero(t, found2.MediaDecodeCapacity)
	assert.Zero(t, found2.MediaEgressCapacityBPS)

	// 5. DeployAlgorithm: offline node (should fail)
	deployReq := dto.DeployAlgorithmRequest{
		AlgoPackageID: "algo-1",
	}
	_, err = svc.DeployAlgorithm(ctx, node.ID, deployReq)
	assert.Error(t, err) // should fail because node is offline

	// Set node online
	found2.Status = model.NodeStatusOnline
	require.NoError(t, db.Save(found2).Error)

	// Deploy nonexistent algorithm (should fail)
	_, err = svc.DeployAlgorithm(ctx, node.ID, deployReq)
	assert.Error(t, err) // should fail because algo does not exist

	// Create algorithm package
	algoPkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "algo-1"},
		AlgorithmName: "yolov8",
		Version:       "1.0.0",
		PackageMD5:    "hash123",
		PackagePath:   "yolov8.tar.gz",
		ExtractPath:   "/opt/aivision/algo/yolov8_1.0.0",
	}
	require.NoError(t, db.Create(algoPkg).Error)

	// Successful deploy trigger
	deployRes, err := svc.DeployAlgorithm(ctx, node.ID, deployReq)
	require.NoError(t, err)
	assert.NotEmpty(t, deployRes.DeploymentID)

	// Retry deploying pending deployment (should fail)
	_, err = svc.DeployAlgorithm(ctx, node.ID, deployReq)
	assert.Error(t, err) // pending deployment exists

	// 6. HandleHeartbeat
	hbReq := &dto.HeartbeatRequest{
		Uptime:        3600,
		CurrentLoad:   2,
		EngineVersion: "1.0.0",
		HALPlatform:   "macos",
		HardwareInfo: dto.HardwareInfo{
			CPUModel:    "Intel i7",
			GPUModel:    "Radeon",
			TotalMemory: 16 * 1024 * 1024 * 1024,
			CPUCores:    8,
		},
	}
	hbRes, err := svc.HandleHeartbeat(ctx, node.ID, hbReq)
	require.NoError(t, err)
	assert.Len(t, hbRes.PendingDeployments, 1)
	assert.Equal(t, "algo-1", hbRes.PendingDeployments[0].AlgoPackageID)
	assert.Equal(t, "http://minio:9000/aivision-algorithms/yolov8.tar.gz", hbRes.PendingDeployments[0].DownloadURL)
	assert.Equal(t, "/opt/aivision/algo/yolov8_1.0.0", hbRes.PendingDeployments[0].ExtractPath)

	// Next heartbeat: sync the installed algorithm
	hbReq2 := &dto.HeartbeatRequest{
		Uptime:        3605,
		CurrentLoad:   2,
		EngineVersion: "1.0.0",
		HALPlatform:   "macos",
		HardwareInfo:  hbReq.HardwareInfo,
		InstalledAlgorithms: []dto.InstalledAlgorithmInfo{
			{
				AlgoPackageID: "algo-1",
				Version:       "1.0.0",
				InstallPath:   "/opt/aivision/algo/yolov8_1.0.0",
				Status:        "installed",
			},
		},
	}
	hbRes2, err := svc.HandleHeartbeat(ctx, node.ID, hbReq2)
	require.NoError(t, err)
	assert.Empty(t, hbRes2.PendingDeployments) // no longer pending

	// Verify installed status in ListAlgorithms
	algos, err := svc.ListAlgorithms(ctx, node.ID)
	require.NoError(t, err)
	assert.Len(t, algos, 1)
	assert.Equal(t, model.AlgoDeployInstalled, algos[0].Status)
	assert.Equal(t, "/opt/aivision/algo/yolov8_1.0.0", algos[0].InstallPath)

	// Deploy already installed algorithm (should fail)
	_, err = svc.DeployAlgorithm(ctx, node.ID, deployReq)
	assert.Error(t, err)

	// 7. Delete Node: with active tasks (should fail)
	task := &model.AIVisionTask{
		BaseModel:    model.BaseModel{ID: "task-1"},
		Name:         "Task 1",
		Status:       model.TaskStatusRunning,
		TargetNodeID: node.ID,
		StartDate:    datatypes.Date(time.Now()),
		EndDate:      datatypes.Date(time.Now().AddDate(0, 0, 7)),
	}
	require.NoError(t, db.Create(task).Error)

	err = svc.Delete(ctx, node.ID)
	assert.Error(t, err) // node has active tasks

	// Set task stopped/error
	task.Status = model.TaskStatusDraft
	require.NoError(t, db.Save(task).Error)

	// Delete Node (should succeed)
	err = svc.Delete(ctx, node.ID)
	require.NoError(t, err)

	// Verify deleted node is not found
	_, err = svc.GetByID(ctx, node.ID)
	assert.Error(t, err)

	// 8. Error paths and coverages
	// SetVersionConfig
	svc.SetVersionConfig("1.0.0", true)

	// Update nonexistent
	err = svc.Update(ctx, "invalid-node-id", updateReq)
	assert.Error(t, err)

	// Update duplicate name
	node2, _, err := svc.Create(ctx, dto.CreateEdgeNodeRequest{
		Name:     "Edge Node 2",
		Endpoint: "http://192.168.1.11:8080",
		MaxLoad:  5,
	})
	require.NoError(t, err)

	err = svc.Update(ctx, node2.ID, dto.UpdateEdgeNodeRequest{Name: "Edge Node 1 Updated"})
	assert.Error(t, err)

	// Deploy nonexistent node
	_, err = svc.DeployAlgorithm(ctx, "invalid-node-id", deployReq)
	assert.Error(t, err)

	// ListAlgorithms nonexistent node
	_, err = svc.ListAlgorithms(ctx, "invalid-node-id")
	assert.Error(t, err)

	// Delete nonexistent node
	err = svc.Delete(ctx, "invalid-node-id")
	assert.Error(t, err)
}

func TestEdgeNodeService_HandleHeartbeat_ResumesSuspendedTasks(t *testing.T) {
	db := setupServiceTestDB(t)

	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	pkgRepo := repository.NewAlgorithmPackageRepository(db)
	taskRepo := repository.NewAIVisionTaskRepository(db)
	jwtManager := jwt.NewManager("my-very-secure-jwt-secret-at-least-32-chars", "niko-admin", "niko-admin", 3600, 86400, nil)

	fileStorage2, _ := storage.NewLocalStorage(".", "http://minio:9000/aivision-algorithms")
	deviceRepo := repository.NewDeviceRepository(db)
	smartRecordRepo := repository.NewSmartRecordRepository(db)
	svc := NewEdgeNodeService(nodeRepo, nodeAlgoRepo, pkgRepo, taskRepo, deviceRepo, smartRecordRepo, jwtManager, fileStorage2, nil)
	ctx := context.Background()

	// Create an online node
	node, token, err := svc.Create(ctx, dto.CreateEdgeNodeRequest{
		Name:     "Suspended State Node",
		Endpoint: "http://192.168.1.20:8080",
		MaxLoad:  5,
	})
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Set node online
	node.Status = model.NodeStatusOnline
	require.NoError(t, db.Save(node).Error)

	// Create suspended tasks for this node
	for i := 0; i < 3; i++ {
		task := &model.AIVisionTask{
			BaseModel:    model.BaseModel{ID: fmt.Sprintf("suspended-task-%d", i)},
			Name:         fmt.Sprintf("Suspended Task %d", i),
			Status:       model.TaskStatusSuspended,
			TargetNodeID: node.ID,
			ErrorReason:  "节点离线导致任务暂停",
			StartDate:    datatypes.Date(time.Now()),
			EndDate:      datatypes.Date(time.Now().AddDate(0, 0, 7)),
		}
		require.NoError(t, db.Create(task).Error)
	}

	// Create a non-suspended task (should NOT be affected by resumption)
	runningTask := &model.AIVisionTask{
		BaseModel:    model.BaseModel{ID: "running-task"},
		Name:         "Running Task",
		Status:       model.TaskStatusRunning,
		TargetNodeID: node.ID,
		StartDate:    datatypes.Date(time.Now()),
		EndDate:      datatypes.Date(time.Now().AddDate(0, 0, 7)),
	}
	require.NoError(t, db.Create(runningTask).Error)

	// Create a task for a different node (should NOT be affected)
	otherNode, _, err := svc.Create(ctx, dto.CreateEdgeNodeRequest{
		Name:     "Other Node",
		Endpoint: "http://192.168.1.30:8080",
		MaxLoad:  5,
	})
	require.NoError(t, err)
	otherNode.Status = model.NodeStatusOnline
	require.NoError(t, db.Save(otherNode).Error)

	otherTask := &model.AIVisionTask{
		BaseModel:    model.BaseModel{ID: "other-suspended-task"},
		Name:         "Other Suspended Task",
		Status:       model.TaskStatusSuspended,
		TargetNodeID: otherNode.ID,
		ErrorReason:  "离线待恢复",
		StartDate:    datatypes.Date(time.Now()),
		EndDate:      datatypes.Date(time.Now().AddDate(0, 0, 7)),
	}
	require.NoError(t, db.Create(otherTask).Error)

	// Send heartbeat to bring node back online and trigger task resumption
	hbReq := &dto.HeartbeatRequest{
		Uptime:        100,
		CurrentLoad:   1,
		EngineVersion: "1.0.0",
		HALPlatform:   "macos",
		HardwareInfo: dto.HardwareInfo{
			CPUModel:    "Apple M1",
			GPUModel:    "Apple M1 GPU",
			TotalMemory: 16 * 1024 * 1024 * 1024,
			CPUCores:    8,
		},
	}
	hbRes, err := svc.HandleHeartbeat(ctx, node.ID, hbReq)
	require.NoError(t, err)
	require.NotNil(t, hbRes)

	// Verify all 3 suspended tasks for this node are now running
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("suspended-task-%d", i)
		task, err := taskRepo.FindByID(ctx, taskID)
		require.NoError(t, err)
		assert.Equal(t, model.TaskStatusRunning, task.Status, "suspended task %d should be resumed", i)
		assert.Empty(t, task.ErrorReason, "error_reason should be cleared for task %d", i)
	}

	// Verify running task is still running (unchanged)
	runningTask2, err := taskRepo.FindByID(ctx, "running-task")
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusRunning, runningTask2.Status)

	// Verify other node's suspended task is NOT affected
	otherTask2, err := taskRepo.FindByID(ctx, "other-suspended-task")
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuspended, otherTask2.Status, "other node's tasks should not be resumed")
	assert.Equal(t, "离线待恢复", otherTask2.ErrorReason, "other node's tasks should retain error reason")
}

func TestEdgeNodeService_buildPresignedURL(t *testing.T) {
	db := setupServiceTestDB(t)

	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	pkgRepo := repository.NewAlgorithmPackageRepository(db)
	taskRepo := repository.NewAIVisionTaskRepository(db)
	jwtManager := jwt.NewManager("my-very-secure-jwt-secret-at-least-32-chars", "niko-admin", "niko-admin", 3600, 86400, nil)

	fileStorage3, _ := storage.NewLocalStorage(".", "http://minio:9000/aivision-algorithms")
	deviceRepo := repository.NewDeviceRepository(db)
	smartRecordRepo := repository.NewSmartRecordRepository(db)
	svc := NewEdgeNodeService(nodeRepo, nodeAlgoRepo, pkgRepo, taskRepo, deviceRepo, smartRecordRepo, jwtManager, fileStorage3, nil)
	ctx := context.Background()

	// Create online node
	node, token, err := svc.Create(ctx, dto.CreateEdgeNodeRequest{
		Name:     "Presigned URL Node",
		Endpoint: "http://192.168.1.40:8080",
		MaxLoad:  5,
	})
	require.NoError(t, err)
	require.NotEmpty(t, token)

	node.Status = model.NodeStatusOnline
	require.NoError(t, db.Save(node).Error)

	// Create two algorithm packages
	for _, ap := range []struct {
		id      string
		name    string
		version string
		path    string
	}{
		{"algo-presigned-1", "face_detection", "1.0.0", "/packages/face_detection_v1.tar.gz"},
		{"algo-presigned-2", "pose_estimation", "2.0.0", "/packages/pose_estimation_v2.tar.gz"},
	} {
		require.NoError(t, db.Create(&model.AlgorithmPackage{
			BaseModel:     model.BaseModel{ID: ap.id},
			AlgorithmName: ap.name,
			Version:       ap.version,
			PackageMD5:    "md5-" + ap.id,
			PackagePath:   ap.path,
			ExtractPath:   "/opt/algo/" + ap.name,
		}).Error)
	}

	// Deploy both algorithms before sending heartbeat
	for _, id := range []string{"algo-presigned-1", "algo-presigned-2"} {
		_, err := svc.DeployAlgorithm(ctx, node.ID, dto.DeployAlgorithmRequest{AlgoPackageID: id})
		require.NoError(t, err)
	}

	// Send heartbeat and verify both URLs
	hbReq := &dto.HeartbeatRequest{
		Uptime:        100,
		CurrentLoad:   1,
		EngineVersion: "1.0.0",
		HALPlatform:   "macos",
		HardwareInfo:  dto.HardwareInfo{CPUModel: "Apple M1"},
	}
	hbRes, err := svc.HandleHeartbeat(ctx, node.ID, hbReq)
	require.NoError(t, err)
	require.Len(t, hbRes.PendingDeployments, 2, "heartbeat should return both pending deployments")

	// Build a map of algo_package_id -> download_url
	urlMap := make(map[string]string)
	for _, d := range hbRes.PendingDeployments {
		urlMap[d.AlgoPackageID] = d.DownloadURL
	}

	// Verify fallback URL construction for each package
	assert.Equal(t, "http://minio:9000/aivision-algorithms/packages/face_detection_v1.tar.gz",
		urlMap["algo-presigned-1"],
		"face_detection URL should use its package path")
	assert.Equal(t, "http://minio:9000/aivision-algorithms/packages/pose_estimation_v2.tar.gz",
		urlMap["algo-presigned-2"],
		"pose_estimation URL should use its package path")

	// Verify the package path has leading / stripped
	assert.NotContains(t, urlMap["algo-presigned-1"], "///", "URL should not have double slashes")
}

func TestEdgeNodeService_PushInferenceResult(t *testing.T) {
	db := setupServiceTestDB(t)

	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	pkgRepo := repository.NewAlgorithmPackageRepository(db)
	taskRepo := repository.NewAIVisionTaskRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	smartRecordRepo := repository.NewSmartRecordRepository(db)
	jwtManager := jwt.NewManager("my-very-secure-jwt-secret-at-least-32-chars", "niko-admin", "niko-admin", 3600, 86400, nil)

	fileStorage, _ := storage.NewLocalStorage(".", "http://minio:9000/aivision-algorithms")
	svc := NewEdgeNodeService(nodeRepo, nodeAlgoRepo, pkgRepo, taskRepo, deviceRepo, smartRecordRepo, jwtManager, fileStorage, nil)

	// Create a camera device
	device := &model.Device{
		BaseModel:  model.BaseModel{ID: "camera-1"},
		DeviceName: "Front Gate Camera",
		AccessType: "rtsp",
		RtspURL:    "rtsp://127.0.0.1/live",
		Status:     "online",
	}
	require.NoError(t, db.Create(device).Error)

	// Create an algorithm package
	algoPkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "algo-presigned-1"},
		AlgorithmName: "face_detection",
		Version:       "1.2.3",
		PackagePath:   "face.tar.gz",
		ExtractPath:   "/opt/algo/face",
		SoPath:        "/opt/algo/face.so",
	}
	require.NoError(t, db.Create(algoPkg).Error)

	// Create an AI task
	task := &model.AIVisionTask{
		BaseModel:       model.BaseModel{ID: "task-1"},
		Name:            "Intrusion Detection Task",
		Status:          model.TaskStatusRunning,
		DeviceChannelID: "camera-1",
		AlgoPackageID:   "algo-presigned-1",
		TargetNodeID:    "node-1",
	}
	require.NoError(t, db.Create(task).Error)

	// Push inference result
	params := &controlproto.InferenceResultParams{
		TaskID:     "task-1",
		AlgoName:   "face_detection",
		DeviceID:   "camera-1",
		FrameTS:    uint64(time.Now().UnixNano()),
		RecordType: "alarm",
		AlarmType:  "intrusion",
		AlarmLevel: "critical",
	}

	svc.PushInferenceResult(params)

	// Wait for worker to flush
	time.Sleep(2200 * time.Millisecond)

	// Verify smart record in DB
	var records []model.SmartRecord
	require.NoError(t, db.Find(&records).Error)
	assert.NotEmpty(t, records)
	assert.Equal(t, "Intrusion Detection Task", records[0].TaskName)
	assert.Equal(t, "Front Gate Camera", records[0].DeviceName)
	assert.Equal(t, "1.2.3", records[0].AlgorithmVersion)
	assert.Equal(t, "intrusion", records[0].AlarmType)
}
