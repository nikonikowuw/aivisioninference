package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
)

func setupTestWebhookHandler() *MediaWebhookHandler {
	// 使用 mock 组件创建 handler
	mockClient := &service.MockEngineClient{}
	logger := zap.NewNop()
	devRepo := repository.NewDeviceRepository(nil)
	mediaStreamRepo := repository.NewMediaStreamRepository(nil)
	stagingRepo := repository.NewDiscoveredDeviceRepository(nil)

	streamManager := service.NewStreamManager(mockClient, devRepo, mediaStreamRepo, logger)
	stagingSvc := service.NewDeviceStagingService(stagingRepo, devRepo, nil, nil)

	// SIPService 不初始化完整 repo，我们用模拟的方式测试 webhook 路由
	// 注意：OnRegister 会调用 sipService.HandleRegister 导致 panic 如果 db 为 nil
	// 因此在测试中 OnRegister 需要在无 sip 环境下运行，或者使用 mock
	return &MediaWebhookHandler{
		mediaService:   &service.MediaService{},
		sipService:     nil,
		streamManager:  streamManager,
		stagingService: stagingSvc,
	}
}

func TestOnRegisterWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestWebhookHandler()

	r := gin.New()
	handler.RegisterRoutes(&r.RouterGroup)

	body, _ := json.Marshal(map[string]interface{}{
		"device_id": "34020000001320000001",
		"remote_ip": "192.168.1.100",
		"port":      5060,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/zlm/callback/on_register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 0, resp.Code)
}

func TestOnPublishWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestWebhookHandler()

	r := gin.New()
	handler.RegisterRoutes(&r.RouterGroup)

	body, _ := json.Marshal(map[string]interface{}{
		"app":    "live",
		"stream": "test-stream",
		"vhost":  "__defaultVhost__",
		"schema": "rtsp",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/zlm/callback/on_publish", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOnPlayWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestWebhookHandler()

	r := gin.New()
	handler.RegisterRoutes(&r.RouterGroup)

	body, _ := json.Marshal(map[string]interface{}{
		"app":    "live",
		"stream": "test-stream",
		"params": "token=test-token",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/zlm/callback/on_play", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOnRecordMP4Webhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestWebhookHandler()

	r := gin.New()
	handler.RegisterRoutes(&r.RouterGroup)

	body, _ := json.Marshal(map[string]interface{}{
		"app":        "record",
		"stream":     "test-stream",
		"file_name":  "test-20240101.mp4",
		"file_path":  "/opt/media/record/test.mp4",
		"file_size":  1024000,
		"start_time": 1704067200,
		"end_time":   1704070800,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/zlm/callback/on_record_mp4", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOnStreamChangedWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestWebhookHandler()

	r := gin.New()
	handler.RegisterRoutes(&r.RouterGroup)

	body, _ := json.Marshal(map[string]interface{}{
		"app":    "live",
		"stream": "test-stream",
		"vhost":  "__defaultVhost__",
		"schema": "rtsp",
		"status": 1,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/zlm/callback/on_stream_changed", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOnStreamNotFoundWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := setupTestWebhookHandler()

	r := gin.New()
	handler.RegisterRoutes(&r.RouterGroup)

	body, _ := json.Marshal(map[string]interface{}{
		"app":    "live",
		"stream": "test-stream",
		"vhost":  "__defaultVhost__",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/zlm/callback/on_stream_not_found", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
