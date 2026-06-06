// Package service 提供业务逻辑层
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// WebhookConfigResponse Webhook 配置响应
type WebhookConfigResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	URL             string          `json:"url"`
	Enabled         bool            `json:"enabled"`
	Tags            []string        `json:"tags"`
	Headers         datatypes.JSON  `json:"headers,omitempty"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	MaxRetries      int             `json:"max_retries"`
	CreatedAt       time.Time       `json:"created_at"`
}

// WebhookCreateRequest Webhook 创建请求
type WebhookCreateRequest struct {
	Name           string            `json:"name" binding:"required"`
	URL            string            `json:"url" binding:"required,url"`
	Enabled        bool              `json:"enabled"`
	Tags           []string          `json:"tags"`
	Headers        map[string]string `json:"headers,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	MaxRetries     int               `json:"max_retries"`
}

// WebhookPushRequest Webhook 推送请求
type WebhookPushRequest struct {
	DeviceSn            string  `json:"deviceSn"`
	CameraCode          string  `json:"cameraCode"`
	CaptureTime         string  `json:"captureTime"`
	SnapshotImagePath   string  `json:"snapshotImagePath"`
	BackgroundImagePath string  `json:"backgroundImagePath"`
	AlarmMajor          string  `json:"alarmMajor"`   // recognition/alarm/detection/ocr/plate
	AlarmType           int     `json:"alarmType"`    // category_code
	Confidence          float64 `json:"confidence"`
}

// WebhookPushLogResponse 推送日志响应
type WebhookPushLogResponse struct {
	ID             string     `json:"id"`
	WebhookID      string     `json:"webhook_id"`
	WebhookName    string     `json:"webhook_name"`
	RequestURL     string     `json:"request_url"`
	ResponseBody   string     `json:"response_body,omitempty"`
	ResponseStatus int        `json:"response_status"`
	Status         string     `json:"status"` // pending/success/failed/retrying
	Attempt        int        `json:"attempt"`
	MaxAttempts    int        `json:"max_attempts"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// WebhookService Webhook 服务
type WebhookService struct {
	db     *gorm.DB
	client *http.Client
}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService(db *gorm.DB) *WebhookService {
	return &WebhookService{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// List 获取 Webhook 列表
func (s *WebhookService) List() ([]WebhookConfigResponse, error) {
	var webhooks []model.WebhookConfig
	if err := s.db.Find(&webhooks).Error; err != nil {
		return nil, err
	}

	result := make([]WebhookConfigResponse, 0, len(webhooks))
	for _, w := range webhooks {
		result = append(result, s.toResponse(w))
	}

	return result, nil
}

// Create 创建 Webhook
func (s *WebhookService) Create(req *WebhookCreateRequest) (*WebhookConfigResponse, error) {
	// 序列化 headers
	headersJSON, _ := json.Marshal(req.Headers)

	webhook := model.WebhookConfig{
		Name:              req.Name,
		URL:               req.URL,
		Enabled:           req.Enabled,
		Headers:           datatypes.JSON(headersJSON),
		PushAlarm:         containsTag(req.Tags, "alarm"),
		PushRecognition:   containsTag(req.Tags, "recognition"),
		PushCapture:       containsTag(req.Tags, "capture"),
		MaxRetries:        req.MaxRetries,
		RetryIntervalBase: 5,
	}

	if webhook.MaxRetries == 0 {
		webhook.MaxRetries = 3
	}

	if err := s.db.Create(&webhook).Error; err != nil {
		return nil, err
	}

	response := s.toResponse(webhook)
	response.Tags = req.Tags
	return &response, nil
}

// GetByID 获取 Webhook 详情
func (s *WebhookService) GetByID(id string) (*WebhookConfigResponse, error) {
	var webhook model.WebhookConfig
	if err := s.db.First(&webhook, "id = ?", id).Error; err != nil {
		return nil, err
	}

	response := s.toResponse(webhook)
	return &response, nil
}

// Update 更新 Webhook
func (s *WebhookService) Update(id string, req *WebhookCreateRequest) (*WebhookConfigResponse, error) {
	var webhook model.WebhookConfig
	if err := s.db.First(&webhook, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 序列化 headers
	headersJSON, _ := json.Marshal(req.Headers)

	webhook.Name = req.Name
	webhook.URL = req.URL
	webhook.Enabled = req.Enabled
	webhook.Headers = datatypes.JSON(headersJSON)
	webhook.PushAlarm = containsTag(req.Tags, "alarm")
	webhook.PushRecognition = containsTag(req.Tags, "recognition")
	webhook.PushCapture = containsTag(req.Tags, "capture")
	webhook.MaxRetries = req.MaxRetries

	if err := s.db.Save(&webhook).Error; err != nil {
		return nil, err
	}

	response := s.toResponse(webhook)
	response.Tags = req.Tags
	return &response, nil
}

// Delete 删除 Webhook
func (s *WebhookService) Delete(id string) error {
	return s.db.Delete(&model.WebhookConfig{}, "id = ?", id).Error
}

var testWebhookPayload = mustMarshal(WebhookPushRequest{
	DeviceSn:            "TEST-DEVICE",
	CameraCode:          "TEST-CAMERA",
	CaptureTime:         "2026-01-01T00:00:00Z",
	SnapshotImagePath:   "/test/snapshot.jpg",
	BackgroundImagePath: "/test/background.jpg",
	AlarmMajor:          "alarm",
	AlarmType:           10004,
	Confidence:          0.95,
})

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// Test 测试 Webhook
func (s *WebhookService) Test(id string) (*WebhookTestResult, error) {
	var webhook model.WebhookConfig
	if err := s.db.First(&webhook, "id = ?", id).Error; err != nil {
		return nil, err
	}

	body := testWebhookPayload

	// 发送请求
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	start := time.Now()
	resp, err := s.sendWebhook(ctx, webhook.URL, webhook.Headers, body)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &WebhookTestResult{
			Success:  false,
			Error:    err.Error(),
			Latency:  latency,
		}, nil
	}

	return &WebhookTestResult{
		Success:      resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode:   resp.StatusCode,
		Latency:      latency,
	}, nil
}

// WebhookTestResult Webhook 测试结果
type WebhookTestResult struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code,omitempty"`
	Latency    int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

// ListLogs 获取推送日志
func (s *WebhookService) ListLogs(webhookID string, page, pageSize int) ([]WebhookPushLogResponse, int64, error) {
	var logs []model.WebhookPushLog
	var total int64

	query := s.db.Model(&model.WebhookPushLog{})
	if webhookID != "" {
		query = query.Where("webhook_id = ?", webhookID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	result := make([]WebhookPushLogResponse, 0, len(logs))
	for _, log := range logs {
		result = append(result, WebhookPushLogResponse{
			ID:             log.ID,
			WebhookID:      log.WebhookID,
			RequestURL:     log.RequestURL,
			ResponseBody:   log.ResponseBody,
			ResponseStatus: log.ResponseStatus,
			Status:         log.Status,
			Attempt:        log.Attempt,
			MaxAttempts:    log.MaxAttempts,
			ErrorMessage:   log.ErrorMessage,
			CreatedAt:      log.CreatedAt,
			CompletedAt:    log.CompletedAt,
		})
	}

	return result, total, nil
}

// PushToMatchingWebhooks 推送到匹配标签的 Webhook
func (s *WebhookService) PushToMatchingWebhooks(ctx context.Context, tags []string, req *WebhookPushRequest) error {
	// 查询所有启用的 Webhook
	var webhooks []model.WebhookConfig
	if err := s.db.Where("enabled = ?", true).Find(&webhooks).Error; err != nil {
		return err
	}

	body, _ := json.Marshal(req)

	// 推送到匹配的 Webhook
	for _, webhook := range webhooks {
		if s.matchTags(webhook, tags) {
			go s.pushWithRetry(ctx, webhook, body)
		}
	}

	return nil
}

// matchTags 检查标签是否匹配
func (s *WebhookService) matchTags(webhook model.WebhookConfig, tags []string) bool {
	// 如果 Webhook 没有配置标签，默认匹配所有
	if webhook.PushAlarm && containsTag(tags, "alarm") {
		return true
	}
	if webhook.PushRecognition && containsTag(tags, "recognition") {
		return true
	}
	if webhook.PushCapture && containsTag(tags, "capture") {
		return true
	}

	// 检查自定义标签
	// TODO: 实现自定义标签匹配

	return false
}

// maxResponseBodySize 响应体最大存储大小 (4KB)，防止数据库被大量响应撑爆
const maxResponseBodySize = 4096

// pushWithRetry 带重试的推送（支持 context 取消，进程退出时可中断）
func (s *WebhookService) pushWithRetry(ctx context.Context, webhook model.WebhookConfig, body []byte) {
	maxRetries := webhook.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	// 重试间隔: 5s, 15s, 30s，超出时使用最大间隔兜底
	retryIntervals := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			zap.L().Debug("webhook push cancelled by context",
				zap.String("webhook_id", webhook.ID),
				zap.String("url", webhook.URL))
			return
		}

		log := s.createPushLog(webhook, body, attempt, maxRetries)

		resp, err := s.sendWebhook(ctx, webhook.URL, webhook.Headers, body)
		if err != nil {
			s.updatePushLogError(&log, err)
			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					return
				case <-time.After(retryInterval(retryIntervals, attempt)):
				}
			}
			continue
		}

		// 限制读取大小，防止大量响应撑爆数据库
		limitedBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
		resp.Body.Close()

		log.ResponseStatus = resp.StatusCode
		log.ResponseBody = string(limitedBody)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Status = model.WebhookPushStatusSuccess
			now := time.Now()
			log.CompletedAt = &now
			s.db.Save(&log)
			return
		}

		log.Status = model.WebhookPushStatusFailed
		log.ErrorMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
		s.db.Save(&log)

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryInterval(retryIntervals, attempt)):
			}
		}
	}
}

// sendWebhook 发送 Webhook 请求
func (s *WebhookService) sendWebhook(ctx context.Context, url string, headers datatypes.JSON, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// 添加自定义请求头
	if headers != nil {
		var headerMap map[string]string
		json.Unmarshal([]byte(headers), &headerMap)
		for k, v := range headerMap {
			req.Header.Set(k, v)
		}
	}

	return s.client.Do(req)
}

// toResponse 转换为响应
func (s *WebhookService) toResponse(webhook model.WebhookConfig) WebhookConfigResponse {
	return WebhookConfigResponse{
		ID:              webhook.ID,
		Name:            webhook.Name,
		URL:             webhook.URL,
		Enabled:         webhook.Enabled,
		Headers:         webhook.Headers,
		TimeoutSeconds:  webhook.MaxRetries * 5, // 简化计算
		MaxRetries:      webhook.MaxRetries,
		CreatedAt:       webhook.CreatedAt,
	}
}

// containsTag 检查标签列表是否包含指定标签
func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func (s *WebhookService) createPushLog(webhook model.WebhookConfig, body []byte, attempt, maxRetries int) model.WebhookPushLog {
	log := model.WebhookPushLog{
		WebhookID:   webhook.ID,
		RequestURL:  webhook.URL,
		RequestBody: datatypes.JSON(body),
		Attempt:     attempt,
		MaxAttempts: maxRetries,
		Status:      model.WebhookPushStatusPending,
	}
	s.db.Create(&log)
	return log
}

func (s *WebhookService) updatePushLogError(log *model.WebhookPushLog, err error) {
	log.Status = model.WebhookPushStatusFailed
	log.ErrorMessage = err.Error()
	s.db.Save(log)
}

func retryInterval(intervals []time.Duration, attempt int) time.Duration {
	idx := attempt - 1
	if idx < len(intervals) {
		return intervals[idx]
	}
	return intervals[len(intervals)-1]
}
