package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/niko-admin/niko-admin/internal/service"
)

func setupTestWebhookHandler() *MediaWebhookHandler {
	return &MediaWebhookHandler{
		mediaService: &service.MediaService{},
		sipService:   &service.SIPService{},
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
