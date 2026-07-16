// Package task 提供基于 Asynq 的后台异步任务队列管理功能
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
)

// 设备状态监控任务类型
const (
	TypeDeviceStatusCheck = "device:status_check" // 设备状态定时检查任务
	TypeDeviceDetect      = "device:detect"       // 设备即时连接探测任务
)

// DeviceStatusCheckPayload 设备状态检查任务载荷
type DeviceStatusCheckPayload struct {
	BatchSize int `json:"batch_size"` // 每批检查的设备数量
}

// DeviceDetectPayload 设备即时探测任务载荷
type DeviceDetectPayload struct {
	ID string `json:"id"` // 设备 ID
}

// DeviceStatusHandler 处理设备状态相关的后台任务
type DeviceStatusHandler struct {
	deviceRepo    *repository.DeviceRepository
	zlmClient     *zlm.Client
	streamManager *service.StreamManager
	edgeNodeRepo  *repository.EdgeNodeRepository
}

// NewDeviceStatusHandler 创建设备状态任务处理器
func NewDeviceStatusHandler(deviceRepo *repository.DeviceRepository, zlmClient *zlm.Client, streamManager *service.StreamManager, edgeNodeRepo *repository.EdgeNodeRepository) *DeviceStatusHandler {
	return &DeviceStatusHandler{
		deviceRepo:    deviceRepo,
		zlmClient:     zlmClient,
		streamManager: streamManager,
		edgeNodeRepo:  edgeNodeRepo,
	}
}

// RegisterHandlers 注册设备状态相关的任务处理函数
func (h *DeviceStatusHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeDeviceStatusCheck, h.handleDeviceStatusCheck)
	mux.HandleFunc(TypeDeviceDetect, h.handleDeviceDetect)
}

// handleDeviceDetect 处理设备即时探测任务
func (h *DeviceStatusHandler) handleDeviceDetect(ctx context.Context, t *asynq.Task) error {
	var payload DeviceDetectPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	if payload.ID == "" {
		return fmt.Errorf("device id is required")
	}

	zap.L().Info("executing instant device detection", zap.String("device_id", payload.ID))

	device, err := h.deviceRepo.FindByID(ctx, payload.ID)
	if err != nil {
		return fmt.Errorf("find device: %w", err)
	}

	// 解析目标节点：优先从已有流状态获取，否则选取任意在线节点
	nodeID := ""
	if h.streamManager != nil {
		if state := h.streamManager.GetStream(ctx, payload.ID); state != nil && state.NodeID != "" {
			nodeID = state.NodeID
		}
	}
	if nodeID == "" && h.edgeNodeRepo != nil {
		node, err := h.edgeNodeRepo.FindAnyOnlineNode(ctx)
		if err == nil && node != nil {
			nodeID = node.ID
		}
	}

	if nodeID == "" {
		// 无可用节点，回退到状态检查
		return h.checkDeviceStatus(ctx, device, &checkStats{})
	}

	metadata := map[string]string{"target_node_id": nodeID}
	err = h.streamManager.Acquire(ctx, payload.ID, "detect", metadata)
	testSuccess := err == nil

	// 探测完成后释放
	defer func() {
		_ = h.streamManager.Release(ctx, payload.ID, "detect")
	}()

	newStatus := model.DeviceStatusOffline
	if testSuccess {
		newStatus = model.DeviceStatusOnline
	}

	if err := h.deviceRepo.UpdateStatus(ctx, payload.ID, newStatus, "", ""); err != nil {
		return fmt.Errorf("update device status: %w", err)
	}

	zap.L().Info("instant device detection completed",
		zap.String("device_id", payload.ID),
		zap.String("status", newStatus))
	return nil
}

// handleDeviceStatusCheck 处理设备状态定时检查任务
// 遍历所有已启用的设备，检查其连接状态并更新
func (h *DeviceStatusHandler) handleDeviceStatusCheck(ctx context.Context, t *asynq.Task) error {
	var payload DeviceStatusCheckPayload
	if len(t.Payload()) > 0 {
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal payload: %w", err)
		}
	}

	batchSize := payload.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	zap.L().Info("starting device status check", zap.Int("batch_size", batchSize))

	// 获取所有已启用的设备
	devices, err := h.deviceRepo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("list enabled devices: %w", err)
	}

	stats := checkStats{
		total:   len(devices),
		checkAt: time.Now(),
	}

	for _, device := range devices {
		if err := h.checkDeviceStatus(ctx, &device, &stats); err != nil {
			zap.L().Error("check device status failed",
				zap.String("device_id", device.ID),
				zap.String("device_name", device.DeviceName),
				zap.Error(err),
			)
		}
	}

	zap.L().Info("device status check completed",
		zap.Int("total", stats.total),
		zap.Int("online", stats.online),
		zap.Int("offline", stats.offline),
		zap.Int("error", stats.errorCount),
		zap.Int("unchanged", stats.unchanged),
		zap.Duration("duration", time.Since(stats.checkAt)),
	)

	return nil
}

// checkStats 统计检查结果
type checkStats struct {
	total      int
	online     int
	offline    int
	errorCount int
	unchanged  int
	checkAt    time.Time
}

// checkDeviceStatus 检查单个设备的状态
func (h *DeviceStatusHandler) checkDeviceStatus(ctx context.Context, device *model.Device, stats *checkStats) error {
	var newStatus string
	var errorCode string
	var errorMessage string

	switch device.AccessType {
	case model.DeviceAccessTypeRTSP:
		newStatus, errorCode, errorMessage = h.checkRTSPDevice(ctx, device)
	case model.DeviceAccessTypeGB28181:
		newStatus, errorCode, errorMessage = h.checkGB28181Device(ctx, device)
	default:
		// 其他类型暂不检查
		stats.unchanged++
		return nil
	}

	// 状态未变化，跳过更新
	if newStatus == device.Status {
		stats.unchanged++
		return nil
	}

	// 更新统计
	switch newStatus {
	case model.DeviceStatusOnline:
		stats.online++
	case model.DeviceStatusOffline:
		stats.offline++
	case model.DeviceStatusError:
		stats.errorCount++
	}

	// 更新设备状态
	if err := h.deviceRepo.UpdateStatus(ctx, device.ID, newStatus, errorCode, errorMessage); err != nil {
		return fmt.Errorf("update device status: %w", err)
	}

	zap.L().Info("device status updated",
		zap.String("device_id", device.ID),
		zap.String("device_name", device.DeviceName),
		zap.String("old_status", device.Status),
		zap.String("new_status", newStatus),
		zap.String("error_message", errorMessage),
	)

	return nil
}

// checkRTSPDevice 检查 RTSP 设备的在线状态
// 通过 ZLM 的 isMediaOnline API 检查流是否在线
func (h *DeviceStatusHandler) checkRTSPDevice(ctx context.Context, device *model.Device) (status, errorCode, errorMessage string) {
	if device.RtspURL == "" {
		return model.DeviceStatusError, "INVALID_CONFIG", "RTSP URL 为空"
	}

	// 构建流标识：使用设备 ID 作为 stream key
	streamKey := fmt.Sprintf("device_%s", device.ID)

	// 检查设备对应的流是否在线
	// 注意：这里假设设备的流已经在 ZLM 中注册（通过 addStreamProxy）
	online, err := h.zlmClient.IsMediaOnline(ctx, "rtsp", "__defaultVhost__", "live", streamKey)
	if err != nil {
		// API 调用失败，可能是 ZLM 服务不可用
		return model.DeviceStatusError, "ZLM_ERROR", fmt.Sprintf("检查流状态失败: %v", err)
	}

	if online {
		return model.DeviceStatusOnline, "", ""
	}

	// 流不在线，但不一定是设备离线（可能是流未被拉取）
	// 检查设备最后在线时间
	if device.LastOnlineAt != nil {
		timeSinceOnline := time.Since(*device.LastOnlineAt)
		if timeSinceOnline < 10*time.Minute {
			// 最近有在线记录，保持在线状态
			return model.DeviceStatusOnline, "", ""
		}
	}

	return model.DeviceStatusOffline, "", ""
}

// checkGB28181Device 检查 GB28181 设备的在线状态
// 基于设备的最后心跳时间判断
func (h *DeviceStatusHandler) checkGB28181Device(ctx context.Context, device *model.Device) (status, errorCode, errorMessage string) {
	if device.GB28181DeviceID == "" {
		return model.DeviceStatusError, "INVALID_CONFIG", "GB28181 设备编码为空"
	}

	// 检查最后在线时间
	if device.LastOnlineAt != nil {
		timeSinceOnline := time.Since(*device.LastOnlineAt)
		// 默认心跳间隔 60 秒，超过 3 倍心跳间隔认为离线
		heartbeatTimeout := 3 * time.Minute

		if timeSinceOnline <= heartbeatTimeout {
			return model.DeviceStatusOnline, "", ""
		}

		return model.DeviceStatusOffline, "", fmt.Sprintf("心跳超时 (%v)", timeSinceOnline.Round(time.Second))
	}

	// 从未在线
	return model.DeviceStatusOffline, "", "设备从未在线"
}
