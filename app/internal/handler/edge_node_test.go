package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:handler_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
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
			deployed_at DATETIME,
			error_message TEXT,
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

	return db
}

func TestEdgeNodeHandler_Endpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerTestDB(t)

	nodeRepo := repository.NewEdgeNodeRepository(db)
	nodeAlgoRepo := repository.NewEdgeNodeAlgorithmRepository(db)
	pkgRepo := repository.NewAlgorithmPackageRepository(db)
	taskRepo := repository.NewAIVisionTaskRepository(db)
	jwtManager := jwt.NewManager("my-very-secure-jwt-secret-at-least-32-chars", "niko-admin", "niko-admin", 3600, 86400, nil)

	fileStorage, _ := storage.NewLocalStorage(".", "http://minio:9000/aivision-algorithms")
	deviceRepo := repository.NewDeviceRepository(db)
	smartRecordRepo := repository.NewSmartRecordRepository(db)
	svc := service.NewEdgeNodeService(nodeRepo, nodeAlgoRepo, pkgRepo, taskRepo, deviceRepo, smartRecordRepo, jwtManager, fileStorage, nil)
	h := NewEdgeNodeHandler(svc)

	r := gin.New()
	// Middleware for i18n or other elements if needed, but simple router works:
	r.POST("/edge-nodes", h.Create)
	r.GET("/edge-nodes", h.List)
	r.GET("/edge-nodes/:id", h.GetByID)
	r.PUT("/edge-nodes/:id", h.Update)
	r.DELETE("/edge-nodes/:id", h.Delete)
	r.POST("/edge-nodes/:id/deploy-algo", h.DeployAlgorithm)
	r.GET("/edge-nodes/:id/algorithms", h.ListAlgorithms)
	r.POST("/edge-nodes/:id/heartbeat", h.Heartbeat)

	// 1. Create Node Endpoint
	createReq := dto.CreateEdgeNodeRequest{
		Name:     "Test Node Handler",
		Endpoint: "http://192.168.1.12:8080",
		MaxLoad:  3,
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/edge-nodes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var createRes struct {
		Code int                    `json:"code"`
		Data CreateEdgeNodeResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createRes))
	assert.Equal(t, 0, createRes.Code)
	nodeID := createRes.Data.Node.ID
	assert.NotEmpty(t, nodeID)
	assert.NotEmpty(t, createRes.Data.Token)

	// Create Invalid (missing fields)
	badCreateReq := dto.CreateEdgeNodeRequest{
		Name: "",
	}
	body, _ = json.Marshal(badCreateReq)
	req = httptest.NewRequest(http.MethodPost, "/edge-nodes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code) // Bad Request from validation

	// 2. GetByID
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/edge-nodes/"+nodeID, nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// GetByID nonexistent
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/edge-nodes/nonexistent-id", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// 3. List
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/edge-nodes?status=offline", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Update
	updateReq := dto.UpdateEdgeNodeRequest{
		Name: "Updated Name",
	}
	body, _ = json.Marshal(updateReq)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/edge-nodes/"+nodeID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. DeployAlgorithm when offline (fails)
	deployReq := dto.DeployAlgorithmRequest{
		AlgoPackageID: "11111111-1111-1111-1111-111111111111",
	}
	body, _ = json.Marshal(deployReq)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/edge-nodes/"+nodeID+"/deploy-algo", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Manually set node online in DB
	require.NoError(t, db.Model(&model.EdgeNode{}).Where("id = ?", nodeID).Update("status", "online").Error)

	// Create algorithm package
	algoPkg := &model.AlgorithmPackage{
		BaseModel:     model.BaseModel{ID: "11111111-1111-1111-1111-111111111111"},
		AlgorithmName: "yolov8",
		Version:       "1.0.0",
		PackageMD5:    "hash123",
		PackagePath:   "yolov8.tar.gz",
	}
	require.NoError(t, db.Create(algoPkg).Error)

	// Deploy (succeeds)
	body, _ = json.Marshal(deployReq)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/edge-nodes/"+nodeID+"/deploy-algo", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. ListAlgorithms
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/edge-nodes/"+nodeID+"/algorithms", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 7. Heartbeat
	hbReq := dto.HeartbeatRequest{
		Uptime:        100,
		CurrentLoad:   1,
		EngineVersion: "1.0.0",
		HALPlatform:   "macos",
		HardwareInfo: dto.HardwareInfo{
			CPUModel:    "Intel",
			CPUCores:    4,
			TotalMemory: 8192,
		},
	}
	body, _ = json.Marshal(hbReq)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/edge-nodes/"+nodeID+"/heartbeat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 8. Delete
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/edge-nodes/"+nodeID, nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
