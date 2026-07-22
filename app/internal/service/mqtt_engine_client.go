package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
)

// MqttEngineClient implements EngineClient using MQTT for communication.
type MqttEngineClient struct {
	mqttClient  mqtt.Client
	syncManager *mqttsync.MqttSyncManager
	logger      *zap.Logger
}

func edgeCommandTopic(nodeID, command string) string {
	return fmt.Sprintf("aivision/edge/%s/cmd/%s", nodeID, command)
}

// NewMqttEngineClient creates a new MqttEngineClient.
func NewMqttEngineClient(
	mqttClient mqtt.Client,
	syncManager *mqttsync.MqttSyncManager,
	logger *zap.Logger,
) *MqttEngineClient {
	return &MqttEngineClient{
		mqttClient:  mqttClient,
		syncManager: syncManager,
		logger:      logger,
	}
}

// ensureClient returns an error if the MQTT client is nil.
func (c *MqttEngineClient) ensureClient() error {
	if c.mqttClient == nil {
		return fmt.Errorf("MQTT client is nil")
	}
	return nil
}

// parseStreamStatus attempts JSON unmarshal first, falling back to FlatBuffers.
func parseStreamStatus(respPayload string) *controlproto.StreamStatusParams {
	var jsStatus struct {
		Status      string `json:"status"`
		PlayURL     string `json:"play_url"`
		ZLMHost     string `json:"zlm_host"`
		ZLMHTTPPort int    `json:"zlm_http_port"`
	}
	if json.Unmarshal([]byte(respPayload), &jsStatus) == nil && jsStatus.Status != "" {
		return &controlproto.StreamStatusParams{
			Status:      jsStatus.Status,
			PlayURL:     jsStatus.PlayURL,
			ZLMHost:     jsStatus.ZLMHost,
			ZLMHTTPPort: jsStatus.ZLMHTTPPort,
		}
	}
	return controlproto.FlatBuffersToStreamStatus([]byte(respPayload))
}

// StartStream publishes a StreamStart command via MQTT.
func (c *MqttEngineClient) StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error) {
	c.logger.Info("MQTT: StartStream", zap.String("node_id", req.NodeID), zap.String("device_id", req.DeviceID))
	if req.NodeID == "" {
		return StreamInfo{}, fmt.Errorf("MQTT: node id is required to start stream")
	}

	traceID := uuid.New().String()
	params := &controlproto.StartStreamParams{
		TaskID:         req.TaskID,
		DeviceID:       req.DeviceID,
		StreamURL:      req.RtspURL,
		DecodeHWType:   0,
		EnableInfer:    req.EnableInfer,
		EnablePlayback: req.EnablePlayback,
		AlgoName:       req.AlgoName,
		AlgoVersion:    req.AlgoVersion,
		SoPath:         req.SoPath,
		AlgoParamsJSON: req.AlgoParamsJSON,
		TraceID:        traceID,
	}

	if params.TaskID == "" {
		params.TaskID = req.DeviceID
	}
	payload, _ := json.Marshal(params)
	topic := edgeCommandTopic(req.NodeID, "start_stream")

	if err := c.ensureClient(); err != nil {
		return StreamInfo{}, err
	}

	token := c.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return StreamInfo{}, token.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 15*time.Second)
	if err != nil {
		return StreamInfo{}, fmt.Errorf("MQTT: wait stream status timeout: %w", err)
	}

	status := parseStreamStatus(respPayload)

	if status == nil || status.Status != "running" {
		return StreamInfo{}, fmt.Errorf("MQTT: engine failed to start stream")
	}

	playURL := status.PlayURL
	if playURL == "" {
		playURL = "rtsp://engine:554/live/" + req.DeviceID
	}

	zlmHost := status.ZLMHost
	zlmHTTPPort := status.ZLMHTTPPort
	if zlmHTTPPort == 0 {
		zlmHTTPPort = 80
	}

	return StreamInfo{
		DeviceID:    req.DeviceID,
		Status:      "active",
		PlayURL:     playURL,
		ZLMHost:     zlmHost,
		ZLMHTTPPort: zlmHTTPPort,
	}, nil
}

// StopStream publishes a StreamStop command via MQTT.
func (c *MqttEngineClient) StopStream(ctx context.Context, nodeID, deviceID string) error {
	c.logger.Info("MQTT: StopStream", zap.String("node_id", nodeID), zap.String("device_id", deviceID))
	if nodeID == "" {
		return fmt.Errorf("MQTT: node id is required to stop stream")
	}

	traceID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{
		"trace_id":  traceID,
		"device_id": deviceID,
	})
	topic := edgeCommandTopic(nodeID, "stop_stream")

	if c.mqttClient == nil {
		return fmt.Errorf("MQTT client is nil")
	}

	token := c.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}

	_, err := c.syncManager.Wait(ctx, traceID, 5*time.Second)
	return err
}

// StartPlayback publishes a StreamPlaybackStart command via MQTT.
func (c *MqttEngineClient) StartPlayback(ctx context.Context, req StreamStartRequest) (string, error) {
	c.logger.Info("MQTT: StartPlayback", zap.String("node_id", req.NodeID), zap.String("device_id", req.DeviceID))
	if req.NodeID == "" {
		return "", fmt.Errorf("MQTT: node id is required to start playback")
	}

	traceID := uuid.New().String()
	params := &controlproto.StartStreamParams{
		TaskID:         req.TaskID,
		DeviceID:       req.DeviceID,
		StreamURL:      req.RtspURL,
		DecodeHWType:   0,
		EnablePlayback: true,
		TraceID:        traceID,
	}

	if params.TaskID == "" {
		params.TaskID = req.DeviceID
	}
	payload, _ := json.Marshal(params)
	topic := edgeCommandTopic(req.NodeID, "start_playback")

	if c.mqttClient == nil {
		return "", fmt.Errorf("MQTT client is nil")
	}

	token := c.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return "", token.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 15*time.Second)
	if err != nil {
		return "", fmt.Errorf("MQTT: wait playback status timeout: %w", err)
	}

	status := parseStreamStatus(respPayload)
	if status == nil || status.Status != "running" {
		return "", fmt.Errorf("MQTT: engine failed to start playback")
	}

	if status.PlayURL != "" {
		return status.PlayURL, nil
	}
	return fmt.Sprintf("rtsp://engine:554/live/%s", req.DeviceID), nil
}

// StopPlayback publishes a StreamPlaybackStop command via MQTT.
func (c *MqttEngineClient) StopPlayback(ctx context.Context, nodeID, deviceID string) error {
	c.logger.Info("MQTT: StopPlayback", zap.String("node_id", nodeID), zap.String("device_id", deviceID))
	if nodeID == "" {
		return fmt.Errorf("MQTT: node id is required to stop playback")
	}

	traceID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{
		"trace_id":  traceID,
		"device_id": deviceID,
	})
	topic := edgeCommandTopic(nodeID, "stop_playback")

	if c.mqttClient == nil {
		return fmt.Errorf("MQTT client is nil")
	}

	token := c.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}

	_, err := c.syncManager.Wait(ctx, traceID, 5*time.Second)
	return err
}

// GetStreamStatus publishes a StreamStatus command via MQTT.
func (c *MqttEngineClient) GetStreamStatus(ctx context.Context, nodeID, deviceID string) (StreamStatus, error) {
	c.logger.Debug("MQTT: GetStreamStatus", zap.String("node_id", nodeID), zap.String("device_id", deviceID))
	if nodeID == "" {
		return StreamStatus{}, fmt.Errorf("MQTT: node id is required to get stream status")
	}

	traceID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{
		"trace_id":  traceID,
		"device_id": deviceID,
	})
	topic := edgeCommandTopic(nodeID, "stream_status")

	if c.mqttClient == nil {
		return StreamStatus{}, fmt.Errorf("MQTT client is nil")
	}

	token := c.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return StreamStatus{}, token.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 5*time.Second)
	if err != nil {
		return StreamStatus{}, fmt.Errorf("MQTT: wait stream status check timeout: %w", err)
	}

	status := parseStreamStatus(respPayload)
	if status != nil {
		return StreamStatus{
			DeviceID:    deviceID,
			Status:      status.Status,
			PlayURL:     status.PlayURL,
			ZLMHost:     status.ZLMHost,
			ZLMHTTPPort: status.ZLMHTTPPort,
		}, nil
	}

	return StreamStatus{
		DeviceID: deviceID,
		Status:   "unknown",
	}, nil
}

// StartSelfCheck publishes a StartSelfCheck command via MQTT.
func (c *MqttEngineClient) StartSelfCheck(ctx context.Context, downloadURL, token, algoName, version string) error {
	c.logger.Info("MQTT: StartSelfCheck", zap.String("algo_name", algoName), zap.String("version", version))

	traceID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{
		"trace_id":     traceID,
		"download_url": downloadURL,
		"token":        token,
		"algo_name":    algoName,
		"version":      version,
	})
	topic := "aivision/edge/self_check/cmd" // Generic topic or node specific if we want

	if err := c.ensureClient(); err != nil {
		return err
	}

	tokenPub := c.mqttClient.Publish(topic, 1, false, payload)
	tokenPub.Wait()
	if tokenPub.Error() != nil {
		return tokenPub.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("MQTT: self check wait timeout: %w", err)
	}

	var result *controlproto.AlgoLoadResultParams
	var jsResult struct {
		Success      bool   `json:"success"`
		ErrorMessage string `json:"error_message"`
		ErrorCode    string `json:"error_code"`
	}
	if json.Unmarshal([]byte(respPayload), &jsResult) == nil {
		result = &controlproto.AlgoLoadResultParams{
			Success:      jsResult.Success,
			ErrorMessage: jsResult.ErrorMessage,
			ErrorCode:    jsResult.ErrorCode,
		}
	}
	if result == nil {
		result = controlproto.FlatBuffersToAlgoLoadResult([]byte(respPayload))
	}

	if result == nil {
		return fmt.Errorf("MQTT: self check response unparseable")
	}

	if !result.Success {
		return fmt.Errorf("MQTT: algorithm self-check failed: %s (error code: %s)", result.ErrorMessage, result.ErrorCode)
	}

	return nil
}

// WarmupAlgorithm asks one node to load a locally installed algorithm runtime.
func (c *MqttEngineClient) WarmupAlgorithm(ctx context.Context, nodeID, algoName, algoVersion string) error {
	c.logger.Info("MQTT: WarmupAlgorithm",
		zap.String("node_id", nodeID),
		zap.String("algo_name", algoName),
		zap.String("algo_version", algoVersion),
	)

	traceID := uuid.New().String()
	payload, err := json.Marshal(struct {
		TraceID     string `json:"trace_id"`
		AlgoName    string `json:"algo_name"`
		AlgoVersion string `json:"algo_version,omitempty"`
	}{
		TraceID:     traceID,
		AlgoName:    algoName,
		AlgoVersion: algoVersion,
	})
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("aivision/edge/%s/cmd/algo_warmup", nodeID)
	if err := c.ensureClient(); err != nil {
		return err
	}

	tokenPub := c.mqttClient.Publish(topic, 1, false, payload)
	tokenPub.Wait()
	if tokenPub.Error() != nil {
		return tokenPub.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 10*time.Second)
	if err != nil {
		return fmt.Errorf("MQTT: algorithm warmup wait timeout: %w", err)
	}

	var result struct {
		Success      bool   `json:"success"`
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal([]byte(respPayload), &result); err != nil {
		return fmt.Errorf("MQTT: algorithm warmup response unparseable: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("MQTT: algorithm warmup failed: %s (error code: %s)", result.ErrorMessage, result.ErrorCode)
	}
	return nil
}

// UpdateFaceLibrary publishes an UpdateFaceLibrary command via MQTT.
func (c *MqttEngineClient) UpdateFaceLibrary(ctx context.Context, nodeID, algoName string, faceLibraryJSON []byte) error {
	c.logger.Info("MQTT: UpdateFaceLibrary", zap.String("node_id", nodeID), zap.String("algo_name", algoName))

	traceID := uuid.New().String()
	payload, err := json.Marshal(struct {
		TraceID         string `json:"trace_id"`
		AlgoName        string `json:"algo_name"`
		FaceLibraryJSON string `json:"face_library_json"`
	}{
		TraceID:         traceID,
		AlgoName:        algoName,
		FaceLibraryJSON: string(faceLibraryJSON),
	})
	if err != nil {
		return err
	}

	topic := fmt.Sprintf("aivision/edge/%s/cmd/face_library", nodeID)
	if err := c.ensureClient(); err != nil {
		return err
	}

	tokenPub := c.mqttClient.Publish(topic, 1, false, payload)
	tokenPub.Wait()
	if tokenPub.Error() != nil {
		return tokenPub.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 10*time.Second)
	if err != nil {
		return fmt.Errorf("MQTT: face library update wait timeout: %w", err)
	}

	var result struct {
		Success      bool   `json:"success"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.Unmarshal([]byte(respPayload), &result); err != nil {
		return fmt.Errorf("MQTT: face library response unparseable: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("MQTT: face library update failed: %s", result.ErrorMessage)
	}

	return nil
}

// ExtractFaceEmbedding publishes an ExtractFaceEmbedding command via MQTT.
func (c *MqttEngineClient) ExtractFaceEmbedding(ctx context.Context, nodeID, algoName, algoVersion string, imageBytes []byte) (FaceEmbeddingResult, error) {
	c.logger.Info("MQTT: ExtractFaceEmbedding", zap.String("node_id", nodeID), zap.String("algo_name", algoName))

	traceID := uuid.New().String()
	payload, err := json.Marshal(struct {
		TraceID     string `json:"trace_id"`
		AlgoName    string `json:"algo_name"`
		AlgoVersion string `json:"algo_version,omitempty"`
		ImageBytes  []byte `json:"image_bytes"`
	}{
		TraceID:     traceID,
		AlgoName:    algoName,
		AlgoVersion: algoVersion,
		ImageBytes:  imageBytes,
	})
	if err != nil {
		return FaceEmbeddingResult{}, err
	}

	topic := fmt.Sprintf("aivision/edge/%s/cmd/face_embedding", nodeID)
	if err := c.ensureClient(); err != nil {
		return FaceEmbeddingResult{}, err
	}

	tokenPub := c.mqttClient.Publish(topic, 1, false, payload)
	tokenPub.Wait()
	if tokenPub.Error() != nil {
		return FaceEmbeddingResult{}, tokenPub.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 30*time.Second)
	if err != nil {
		return FaceEmbeddingResult{}, fmt.Errorf("MQTT: face embedding wait timeout: %w", err)
	}

	var result FaceEmbeddingResult
	if err := json.Unmarshal([]byte(respPayload), &result); err != nil {
		return FaceEmbeddingResult{}, fmt.Errorf("MQTT: face embedding response unparseable: %w", err)
	}

	if !result.Success {
		return result, fmt.Errorf("MQTT: face embedding extraction failed: %s", result.ErrorMessage)
	}

	return result, nil
}
