package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
)

// EdgeMqttHandler processes MQTT messages received from the edge nodes.
type EdgeMqttHandler struct {
	nodeSvc              *service.EdgeNodeService
	syncManager           *mqttsync.MqttSyncManager
	taskClient            *task.Client
	rdb                   *redis.Client
	hub                   *ws.Hub
	metricsStore          *service.EngineMetricsStore
	scheduledTaskSvc      *service.EdgeNodeScheduledTaskService
	terminalSvc           *service.EdgeNodeTerminalService
}

// NewEdgeMqttHandler creates a new EdgeMqttHandler.
func NewEdgeMqttHandler(
	nodeSvc *service.EdgeNodeService,
	syncManager *mqttsync.MqttSyncManager,
	taskClient *task.Client,
	rdb *redis.Client,
	hub *ws.Hub,
	metricsStore *service.EngineMetricsStore,
	scheduledTaskSvc *service.EdgeNodeScheduledTaskService,
	terminalSvc *service.EdgeNodeTerminalService,
) *EdgeMqttHandler {
	return &EdgeMqttHandler{
		nodeSvc:         nodeSvc,
		syncManager:     syncManager,
		taskClient:      taskClient,
		rdb:             rdb,
		hub:             hub,
		metricsStore:    metricsStore,
		scheduledTaskSvc: scheduledTaskSvc,
		terminalSvc:     terminalSvc,
	}
}

// HandleEngineMetrics stores the latest media and inference metrics for one node.
func (h *EdgeMqttHandler) HandleEngineMetrics(msg mqtt.Message) {
	topicParts := strings.Split(msg.Topic(), "/")
	if len(topicParts) < 3 || h.metricsStore == nil {
		return
	}
	nodeID := topicParts[2]
	snapshot := controlproto.FlatBuffersToEngineMetrics(msg.Payload())
	if snapshot == nil {
		zap.L().Warn("MQTT: invalid engine metrics payload", zap.String("node_id", nodeID))
		return
	}
	h.metricsStore.UpdateNode(nodeID, snapshot)
}

// HandleHeartbeat processes a heartbeat message from an edge node.
func (h *EdgeMqttHandler) HandleHeartbeat(msg mqtt.Message) {
	topicParts := strings.Split(msg.Topic(), "/")
	if len(topicParts) < 3 {
		return
	}
	nodeID := topicParts[2]

	var req dto.HeartbeatRequest
	if err := json.Unmarshal(msg.Payload(), &req); err != nil {
		zap.L().Error("MQTT: failed to parse heartbeat payload", zap.String("node_id", nodeID), zap.Error(err))
		return
	}

	// Process heartbeat in the EdgeNodeService (db status, uptime updates, etc.)
	hbCtx, hbCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer hbCancel()
	_, err := h.nodeSvc.HandleHeartbeat(hbCtx, nodeID, &req)
	if err != nil {
		zap.L().Error("MQTT: HandleHeartbeat in service failed", zap.String("node_id", nodeID), zap.Error(err))
		return
	}

	// State reconciliation detection
	sort.Strings(req.ActiveStreams)
	joinedStreams := strings.Join(req.ActiveStreams, ",")
	hasher := md5.New()
	hasher.Write([]byte(joinedStreams))
	hash := hex.EncodeToString(hasher.Sum(nil))

	hashKey := fmt.Sprintf("aivision:edge:%s:state_hash", nodeID)
	oldHash, err := h.rdb.Get(context.Background(), hashKey).Result()

	if err == redis.Nil || oldHash != hash {
		zap.L().Info("MQTT: active streams hash changed, enqueuing reconciliation",
			zap.String("node_id", nodeID), zap.String("old", oldHash), zap.String("new", hash))

		// Store hash in Redis with a 5-minute TTL
		if err := h.rdb.Set(context.Background(), hashKey, hash, 5*time.Minute).Err(); err != nil {
			zap.L().Error("MQTT: failed to set active streams hash in Redis", zap.Error(err))
		}

		// Store actual streams for the Asynq worker to inspect
		streamsKey := fmt.Sprintf("aivision:edge:%s:actual_streams", nodeID)
		streamsBytes, _ := json.Marshal(req.ActiveStreams)
		if err := h.rdb.Set(context.Background(), streamsKey, streamsBytes, 1*time.Hour).Err(); err != nil {
			zap.L().Error("MQTT: failed to set actual active streams in Redis", zap.Error(err))
		}

		// Enqueue the state reconciliation worker
		payload := map[string]interface{}{
			"node_id": nodeID,
		}
		if err := h.taskClient.Enqueue(context.Background(), task.TaskReconcileEdgeState, payload); err != nil {
			zap.L().Error("MQTT: failed to enqueue state reconciliation task", zap.Error(err))
		}
	}
}

// HandleStreamStatus processes stream start/stop response messages from an edge node.
func (h *EdgeMqttHandler) HandleStreamStatus(msg mqtt.Message) {
	var payload struct {
		TraceID string `json:"trace_id"`
	}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		zap.L().Error("MQTT: failed to parse stream status response payload", zap.Error(err))
		return
	}

	if payload.TraceID != "" {
		zap.L().Info("MQTT: resolving stream status trace_id", zap.String("trace_id", payload.TraceID))
		if err := h.syncManager.Resolve(context.Background(), payload.TraceID, string(msg.Payload())); err != nil {
			zap.L().Error("MQTT: failed to resolve sync trace_id", zap.String("trace_id", payload.TraceID), zap.Error(err))
		}
	}
}

// HandleInferenceResult processes real-time bounding box outputs from an edge node.
func (h *EdgeMqttHandler) HandleInferenceResult(msg mqtt.Message) {
	params := controlproto.FlatBuffersToInferenceResult(msg.Payload())
	if params == nil {
		return
	}

	if params.DeviceID == "" {
		zap.L().Warn("MQTT: inference result with empty DeviceID, skipping")
		return
	}

	h.nodeSvc.PushInferenceResult(params)

	if h.hub != nil {
		h.hub.Broadcast(&ws.Message{
			Type:     ws.TopicInference,
			DeviceID: params.DeviceID,
			Payload: map[string]interface{}{
				"device_id":  params.DeviceID,
				"task_id":    params.TaskID,
				"detections": params.Detections,
			},
		})
	}
}

// HandleLifecycle processes Last Will and Testament (LWT) messages for device status updates.
func (h *EdgeMqttHandler) HandleLifecycle(msg mqtt.Message) {
	topicParts := strings.Split(msg.Topic(), "/")
	if len(topicParts) < 3 {
		return
	}
	nodeID := topicParts[2]

	payloadStr := strings.TrimSpace(string(msg.Payload()))
	if payloadStr == "offline" {
		zap.L().Warn("MQTT: edge node lifecycle offline received, processing offline",
			zap.String("node_id", nodeID))

		if err := h.nodeSvc.HandleLWTNodeOffline(context.Background(), nodeID); err != nil {
			zap.L().Error("MQTT: HandleLWTNodeOffline failed",
				zap.String("node_id", nodeID),
				zap.Error(err),
			)
			return
		}

		zap.L().Warn("MQTT: edge node went offline via LWT",
			zap.String("node_id", nodeID))
	}
}

// HandleShellExecResult processes shell_exec_result responses from the edge node.
func (h *EdgeMqttHandler) HandleShellExecResult(msg mqtt.Message) {
	topicParts := strings.Split(msg.Topic(), "/")
	if len(topicParts) < 3 {
		return
	}
	nodeID := topicParts[2]

	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		zap.L().Error("MQTT: failed to parse shell_exec_result payload",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return
	}

	if h.scheduledTaskSvc != nil {
		h.scheduledTaskSvc.HandleShellExecResult(context.Background(), nodeID, payload)
	}
}

// HandlePtyOutput processes pty_output responses from the edge node.
func (h *EdgeMqttHandler) HandlePtyOutput(msg mqtt.Message) {
	topicParts := strings.Split(msg.Topic(), "/")
	if len(topicParts) < 3 {
		return
	}
	nodeID := topicParts[2]

	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		zap.L().Error("MQTT: failed to parse pty_output payload",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return
	}

	if h.terminalSvc != nil {
		h.terminalSvc.HandlePtyOutput(nodeID, payload)
	}
}

// HandlePtyError processes pty_error responses from the edge node.
func (h *EdgeMqttHandler) HandlePtyError(msg mqtt.Message) {
	topicParts := strings.Split(msg.Topic(), "/")
	if len(topicParts) < 3 {
		return
	}
	nodeID := topicParts[2]

	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		zap.L().Error("MQTT: failed to parse pty_error payload",
			zap.String("node_id", nodeID),
			zap.Error(err),
		)
		return
	}

	if h.terminalSvc != nil {
		h.terminalSvc.HandlePtyError(nodeID, payload)
	}
}
