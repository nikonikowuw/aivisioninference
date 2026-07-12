package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// EdgeScheduledTaskService 处理计划任务的业务逻辑
type EdgeScheduledTaskService struct {
	taskRepo       *repository.EdgeScheduledTaskRepository
	recordRepo     *repository.EdgeScheduledTaskRecordRepository
	tagRepo        *repository.EdgeNodeTagRepository
	nodeRepo       *repository.EdgeNodeRepository
	mqttClient     mqtt.Client
	syncManager    *mqttsync.MqttSyncManager
	hub            *ws.Hub
}

// NewEdgeScheduledTaskService 创建新的 EdgeScheduledTaskService
func NewEdgeScheduledTaskService(
	taskRepo *repository.EdgeScheduledTaskRepository,
	recordRepo *repository.EdgeScheduledTaskRecordRepository,
	tagRepo *repository.EdgeNodeTagRepository,
	nodeRepo *repository.EdgeNodeRepository,
	mqttClient mqtt.Client,
	syncManager *mqttsync.MqttSyncManager,
	hub *ws.Hub,
) *EdgeScheduledTaskService {
	return &EdgeScheduledTaskService{
		taskRepo:    taskRepo,
		recordRepo:  recordRepo,
		tagRepo:     tagRepo,
		nodeRepo:    nodeRepo,
		mqttClient:  mqttClient,
		syncManager: syncManager,
		hub:         hub,
	}
}

// validCommandNames 返回支持的命令名称集合
var validCommandNames = map[string]bool{
	model.ScheduledTaskCmdStartStream:   true,
	model.ScheduledTaskCmdStopStream:    true,
	model.ScheduledTaskCmdStartPlayback: true,
	model.ScheduledTaskCmdStopPlayback:  true,
	model.ScheduledTaskCmdStreamStatus:  true,
	model.ScheduledTaskCmdSelfCheck:     true,
	model.ScheduledTaskCmdFaceLibrary:   true,
	model.ScheduledTaskCmdAlgoWarmup:    true,
	model.ScheduledTaskCmdFaceEmbedding: true,
}

// IsValidCommandName 检查命令名称是否合法
func IsValidCommandName(name string) bool {
	return validCommandNames[name]
}

// Create 创建计划任务
func (s *EdgeScheduledTaskService) Create(ctx context.Context, req dto.CreateEdgeScheduledTaskRequest) (*model.EdgeScheduledTask, error) {
	if !IsValidCommandName(req.CommandName) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "不支持的命令名称")
	}

	paramsJSON := "{}"
	if req.CommandParams != nil {
		b, err := json.Marshal(req.CommandParams)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrBadRequest, "命令参数格式错误")
		}
		paramsJSON = string(b)
	}

	item := &model.EdgeScheduledTask{
		Name:             req.Name,
		Description:      req.Description,
		CronExpr:         req.CronExpr,
		TargetType:       req.TargetType,
		TargetID:         req.TargetID,
		CommandName:      req.CommandName,
		CommandParams:    []byte(paramsJSON),
		WaitResponse:     req.WaitResponse,
		WaitTimeoutSec:   req.WaitTimeoutSec,
		MaxRetries:       req.MaxRetries,
		RetryIntervalSec: req.RetryIntervalSec,
		Enabled:          true,
	}
	if item.WaitTimeoutSec <= 0 {
		item.WaitTimeoutSec = 30
	}
	if item.MaxRetries <= 0 {
		item.MaxRetries = 3
	}
	if item.RetryIntervalSec <= 0 {
		item.RetryIntervalSec = 60
	}

	if err := s.taskRepo.Create(ctx, item); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, apperrors.New(apperrors.ErrBadRequest, "任务名称已存在")
		}
		return nil, err
	}
	return item, nil
}

// Update 更新计划任务
func (s *EdgeScheduledTaskService) Update(ctx context.Context, id string, req dto.UpdateEdgeScheduledTaskRequest) error {
	item, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.New(apperrors.ErrNotFound, "计划任务不存在")
		}
		return err
	}

	if !IsValidCommandName(req.CommandName) {
		return apperrors.New(apperrors.ErrBadRequest, "不支持的命令名称")
	}

	paramsJSON := "{}"
	if req.CommandParams != nil {
		b, err := json.Marshal(req.CommandParams)
		if err != nil {
			return apperrors.New(apperrors.ErrBadRequest, "命令参数格式错误")
		}
		paramsJSON = string(b)
	}

	item.Name = req.Name
	item.Description = req.Description
	item.CronExpr = req.CronExpr
	item.TargetType = req.TargetType
	item.TargetID = req.TargetID
	item.CommandName = req.CommandName
	item.CommandParams = []byte(paramsJSON)
	item.WaitResponse = req.WaitResponse
	item.WaitTimeoutSec = req.WaitTimeoutSec
	item.MaxRetries = req.MaxRetries
	item.RetryIntervalSec = req.RetryIntervalSec
	if item.WaitTimeoutSec <= 0 {
		item.WaitTimeoutSec = 30
	}
	if item.MaxRetries <= 0 {
		item.MaxRetries = 3
	}
	if item.RetryIntervalSec <= 0 {
		item.RetryIntervalSec = 60
	}

	return s.taskRepo.Update(ctx, item)
}

// Delete 删除计划任务
func (s *EdgeScheduledTaskService) Delete(ctx context.Context, id string) error {
	_, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.New(apperrors.ErrNotFound, "计划任务不存在")
		}
		return err
	}
	return s.taskRepo.Delete(ctx, id)
}

// GetByID 查询计划任务
func (s *EdgeScheduledTaskService) GetByID(ctx context.Context, id string) (*model.EdgeScheduledTask, error) {
	item, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(apperrors.ErrNotFound, "计划任务不存在")
		}
		return nil, err
	}
	return item, nil
}

// List 分页查询计划任务列表
func (s *EdgeScheduledTaskService) List(ctx context.Context, req dto.EdgeScheduledTaskListRequest) ([]model.EdgeScheduledTask, int64, error) {
	return s.taskRepo.List(ctx, req)
}

// ToggleEnabled 启用/禁用计划任务
func (s *EdgeScheduledTaskService) ToggleEnabled(ctx context.Context, id string) (*model.EdgeScheduledTask, error) {
	item, err := s.taskRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(apperrors.ErrNotFound, "计划任务不存在")
		}
		return nil, err
	}
	item.Enabled = !item.Enabled
	if err := s.taskRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// resolveTargetNodes 解析目标节点 ID 列表
func (s *EdgeScheduledTaskService) resolveTargetNodes(ctx context.Context, task *model.EdgeScheduledTask) ([]string, error) {
	switch task.TargetType {
	case "single_node":
		return []string{task.TargetID}, nil
	case "tag":
		return s.tagRepo.FindNodesByTagID(ctx, task.TargetID)
	default:
		return nil, fmt.Errorf("不支持的目标类型: %s", task.TargetType)
	}
}

// DispatchCommand 向指定节点下发 MQTT 命令
func (s *EdgeScheduledTaskService) DispatchCommand(ctx context.Context, nodeID, commandName string, params map[string]interface{}, waitResponse bool, timeoutSec int) (traceID string, respPayload string, err error) {
	traceID = uuid.New().String()
	payload := map[string]interface{}{
		"trace_id": traceID,
	}
	for k, v := range params {
		payload[k] = v
	}

	payloadBytes, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return "", "", fmt.Errorf("序列化命令参数失败: %w", marshalErr)
	}

	topic := edgeCommandTopic(nodeID, commandName)

	if s.mqttClient == nil {
		return "", "", fmt.Errorf("MQTT client is nil")
	}

	token := s.mqttClient.Publish(topic, 1, false, payloadBytes)
	token.Wait()
	if token.Error() != nil {
		return traceID, "", fmt.Errorf("MQTT 发布失败: %w", token.Error())
	}

	if waitResponse {
		timeout := time.Duration(timeoutSec) * time.Second
		resp, waitErr := s.syncManager.Wait(ctx, traceID, timeout)
		if waitErr != nil {
			return traceID, "", fmt.Errorf("等待响应超时: %w", waitErr)
		}
		return traceID, resp, nil
	}

	return traceID, "", nil
}

// ExecuteTask 执行计划任务
func (s *EdgeScheduledTaskService) ExecuteTask(ctx context.Context, task *model.EdgeScheduledTask) {
	logger := zap.L().With(
		zap.String("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.String("command", task.CommandName),
	)

	nodeIDs, err := s.resolveTargetNodes(ctx, task)
	if err != nil {
		logger.Error("解析目标节点失败", zap.Error(err))
		return
	}

	var params map[string]interface{}
	if len(task.CommandParams) > 0 {
		if err := json.Unmarshal(task.CommandParams, &params); err != nil {
			logger.Error("解析命令参数失败", zap.Error(err))
			params = make(map[string]interface{})
		}
	} else {
		params = make(map[string]interface{})
	}

	for _, nodeID := range nodeIDs {
		record := &model.EdgeScheduledTaskRecord{
			TaskID:         task.ID,
			NodeID:         nodeID,
			Status:         model.ScheduledTaskRecordStatusPending,
			RequestPayload: marshalPayload(params),
		}
		if err := s.recordRepo.Create(ctx, record); err != nil {
			logger.Error("创建执行记录失败", zap.String("node_id", nodeID), zap.Error(err))
			continue
		}

		now := time.Now()
		record.ExecutedAt = &now

		traceID, respPayload, dispatchErr := s.DispatchCommand(ctx, nodeID, task.CommandName, params, task.WaitResponse, task.WaitTimeoutSec)
		record.TraceID = traceID

		if dispatchErr != nil {
			record.Status = model.ScheduledTaskRecordStatusFailed
			record.ErrorMessage = dispatchErr.Error()
			completed := time.Now()
			record.CompletedAt = &completed
			logger.Warn("命令下发失败",
				zap.String("node_id", nodeID),
				zap.String("record_id", record.ID),
				zap.Error(dispatchErr),
			)
		} else if task.WaitResponse {
			record.Status = model.ScheduledTaskRecordStatusSuccess
			if respPayload != "" {
				record.ResponsePayload = []byte(respPayload)
			}
			completed := time.Now()
			record.CompletedAt = &completed
		} else {
			record.Status = model.ScheduledTaskRecordStatusDispatched
		}

		if updateErr := s.recordRepo.Update(ctx, record); updateErr != nil {
			logger.Error("更新执行记录失败", zap.String("record_id", record.ID), zap.Error(updateErr))
		}

		// WebSocket 广播
		s.broadcastExecutionEvent(task, record)
	}
}

// RetryRecord 重试失败的计划任务执行记录
func (s *EdgeScheduledTaskService) RetryRecord(ctx context.Context, recordID string) (*model.EdgeScheduledTaskRecord, error) {
	record, err := s.recordRepo.FindByID(ctx, recordID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(apperrors.ErrNotFound, "执行记录不存在")
		}
		return nil, err
	}

	if record.Status != model.ScheduledTaskRecordStatusFailed {
		return nil, apperrors.New(apperrors.ErrBadRequest, "只能重试失败的记录")
	}

	task, err := s.taskRepo.FindByID(ctx, record.TaskID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "计划任务不存在")
	}

	// 重置状态
	record.RetryCount++
	record.Status = model.ScheduledTaskRecordStatusPending
	record.ErrorMessage = ""
	record.ResponsePayload = nil
	now := time.Now()
	record.ExecutedAt = &now
	record.CompletedAt = nil

	if err := s.recordRepo.Update(ctx, record); err != nil {
		return nil, err
	}

	var params map[string]interface{}
	if len(task.CommandParams) > 0 {
		json.Unmarshal(task.CommandParams, &params)
	}

	traceID, respPayload, dispatchErr := s.DispatchCommand(ctx, record.NodeID, task.CommandName, params, task.WaitResponse, task.WaitTimeoutSec)
	record.TraceID = traceID

	if dispatchErr != nil {
		record.Status = model.ScheduledTaskRecordStatusFailed
		record.ErrorMessage = dispatchErr.Error()
		completed := time.Now()
		record.CompletedAt = &completed
	} else if task.WaitResponse {
		record.Status = model.ScheduledTaskRecordStatusSuccess
		if respPayload != "" {
			record.ResponsePayload = []byte(respPayload)
		}
		completed := time.Now()
		record.CompletedAt = &completed
	} else {
		record.Status = model.ScheduledTaskRecordStatusDispatched
	}

	if updateErr := s.recordRepo.Update(ctx, record); updateErr != nil {
		return nil, updateErr
	}

	s.broadcastExecutionEvent(task, record)
	return record, nil
}

// CleanupRecords 清理过期执行记录
func (s *EdgeScheduledTaskService) CleanupRecords(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -90)
	deleted, err := s.recordRepo.CleanupOlderThan(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

// ListRecords 查询执行记录
func (s *EdgeScheduledTaskService) ListRecords(ctx context.Context, req dto.EdgeScheduledTaskRecordListRequest) ([]model.EdgeScheduledTaskRecord, int64, error) {
	return s.recordRepo.List(ctx, req)
}

// GetRecordByID 查询单条执行记录
func (s *EdgeScheduledTaskService) GetRecordByID(ctx context.Context, id string) (*model.EdgeScheduledTaskRecord, error) {
	record, err := s.recordRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(apperrors.ErrNotFound, "执行记录不存在")
		}
		return nil, err
	}
	return record, nil
}

// ListEnabledTasks 查询所有启用的计划任务
func (s *EdgeScheduledTaskService) ListEnabledTasks(ctx context.Context) ([]model.EdgeScheduledTask, error) {
	return s.taskRepo.ListEnabled(ctx)
}

func (s *EdgeScheduledTaskService) broadcastExecutionEvent(task *model.EdgeScheduledTask, record *model.EdgeScheduledTaskRecord) {
	if s.hub == nil {
		return
	}

	eventType := "scheduled_task.executed"
	if record.Status == model.ScheduledTaskRecordStatusSuccess || record.Status == model.ScheduledTaskRecordStatusFailed {
		eventType = "scheduled_task.completed"
	}

	s.hub.Broadcast(&ws.Message{
		Type: eventType,
		Payload: map[string]interface{}{
			"task_id":      task.ID,
			"task_name":    task.Name,
			"record_id":    record.ID,
			"node_id":      record.NodeID,
			"command_name": task.CommandName,
			"status":       record.Status,
			"executed_at":  record.ExecutedAt,
			"completed_at": record.CompletedAt,
		},
	})
}

// marshalPayload 将 map 序列化为 json.RawMessage，失败时返回空 JSON 对象
func marshalPayload(params map[string]interface{}) []byte {
	b, err := json.Marshal(params)
	if err != nil {
		return []byte("{}")
	}
	return b
}
