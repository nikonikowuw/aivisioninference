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
	"github.com/niko-admin/niko-admin/internal/pkg/ipc"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
)

// EdgeMqttHandler processes MQTT messages received from the edge nodes.
type EdgeMqttHandler struct {
	nodeSvc     *service.EdgeNodeService
	syncManager *mqttsync.MqttSyncManager
	taskClient  *task.Client
	rdb         *redis.Client
	hub         *ws.Hub
}

// NewEdgeMqttHandler creates a new EdgeMqttHandler.
func NewEdgeMqttHandler(
	nodeSvc *service.EdgeNodeService,
	syncManager *mqttsync.MqttSyncManager,
	taskClient *task.Client,
	rdb *redis.Client,
	hub *ws.Hub,
) *EdgeMqttHandler {
	return &EdgeMqttHandler{
		nodeSvc:     nodeSvc,
		syncManager: syncManager,
		taskClient:  taskClient,
		rdb:         rdb,
		hub:         hub,
	}
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
	_, err := h.nodeSvc.HandleHeartbeat(context.Background(), nodeID, &req)
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
	params := ipc.FlatBuffersToInferenceResult(msg.Payload())
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
			Type:     "inference",
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
		zap.L().Warn("MQTT: edge node lifecycle offline received, updating status to offline", zap.String("node_id", nodeID))
		var req dto.UpdateEdgeNodeRequest
		status := "offline"
		req.Status = status
		if err := h.nodeSvc.Update(context.Background(), nodeID, req); err != nil {
			zap.L().Error("MQTT: failed to update edge node status to offline", zap.String("node_id", nodeID), zap.Error(err))
		}
	}
}
