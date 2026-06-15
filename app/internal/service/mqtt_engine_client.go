package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/pkg/ipc"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
)

// MqttEngineClient implements EngineClient using MQTT for communication.
type MqttEngineClient struct {
	mqttClient  mqtt.Client
	syncManager *mqttsync.MqttSyncManager
	logger      *zap.Logger
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

// StartStream publishes a StreamStart command via MQTT.
func (c *MqttEngineClient) StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error) {
	c.logger.Info("MQTT: StartStream", zap.String("device_id", req.DeviceID))

	traceID := uuid.New().String()
	params := &ipc.StartStreamParams{
		TaskID:         req.DeviceID,
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

	payload, _ := json.Marshal(params)
	topic := fmt.Sprintf("aivision/edge/%s/cmd/start_stream", req.DeviceID)

	if c.mqttClient == nil {
		return StreamInfo{}, fmt.Errorf("MQTT client is nil")
	}

	token := c.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return StreamInfo{}, token.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 5*time.Second)
	if err != nil {
		return StreamInfo{}, fmt.Errorf("MQTT: wait stream status timeout: %w", err)
	}

	var status *ipc.StreamStatusParams
	var jsStatus struct {
		Status  string `json:"status"`
		PlayURL string `json:"play_url"`
	}
	if json.Unmarshal([]byte(respPayload), &jsStatus) == nil && jsStatus.Status != "" {
		status = &ipc.StreamStatusParams{
			Status:  jsStatus.Status,
			PlayURL: jsStatus.PlayURL,
		}
	}
	if status == nil {
		status = ipc.FlatBuffersToStreamStatus([]byte(respPayload))
	}

	if status == nil || status.Status != "running" {
		return StreamInfo{}, fmt.Errorf("MQTT: engine failed to start stream")
	}

	playURL := status.PlayURL
	if playURL == "" {
		playURL = "rtsp://engine:554/live/" + req.DeviceID
	}

	return StreamInfo{
		DeviceID: req.DeviceID,
		Status:   "active",
		PlayURL:  playURL,
	}, nil
}

// StopStream publishes a StreamStop command via MQTT.
func (c *MqttEngineClient) StopStream(ctx context.Context, deviceID string) error {
	c.logger.Info("MQTT: StopStream", zap.String("device_id", deviceID))

	traceID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{
		"trace_id":  traceID,
		"device_id": deviceID,
	})
	topic := fmt.Sprintf("aivision/edge/%s/cmd/stop_stream", deviceID)

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
	c.logger.Info("MQTT: StartPlayback", zap.String("device_id", req.DeviceID))

	traceID := uuid.New().String()
	params := &ipc.StartStreamParams{
		TaskID:         req.DeviceID,
		DeviceID:       req.DeviceID,
		StreamURL:      req.RtspURL,
		DecodeHWType:   0,
		EnablePlayback: true,
		TraceID:        traceID,
	}

	payload, _ := json.Marshal(params)
	topic := fmt.Sprintf("aivision/edge/%s/cmd/start_playback", req.DeviceID)

	if c.mqttClient == nil {
		return "", fmt.Errorf("MQTT client is nil")
	}

	token := c.mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return "", token.Error()
	}

	respPayload, err := c.syncManager.Wait(ctx, traceID, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("MQTT: wait playback status timeout: %w", err)
	}

	var status *ipc.StreamStatusParams
	var jsStatus struct {
		Status  string `json:"status"`
		PlayURL string `json:"play_url"`
	}
	if json.Unmarshal([]byte(respPayload), &jsStatus) == nil && jsStatus.Status != "" {
		status = &ipc.StreamStatusParams{
			Status:  jsStatus.Status,
			PlayURL: jsStatus.PlayURL,
		}
	}
	if status == nil {
		status = ipc.FlatBuffersToStreamStatus([]byte(respPayload))
	}

	if status == nil || status.Status != "running" {
		return "", fmt.Errorf("MQTT: engine failed to start playback")
	}

	if status.PlayURL != "" {
		return status.PlayURL, nil
	}
	return fmt.Sprintf("rtsp://engine:554/live/%s", req.DeviceID), nil
}

// StopPlayback publishes a StreamPlaybackStop command via MQTT.
func (c *MqttEngineClient) StopPlayback(ctx context.Context, deviceID string) error {
	c.logger.Info("MQTT: StopPlayback", zap.String("device_id", deviceID))

	traceID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{
		"trace_id":  traceID,
		"device_id": deviceID,
	})
	topic := fmt.Sprintf("aivision/edge/%s/cmd/stop_playback", deviceID)

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
func (c *MqttEngineClient) GetStreamStatus(ctx context.Context, deviceID string) (StreamStatus, error) {
	c.logger.Debug("MQTT: GetStreamStatus", zap.String("device_id", deviceID))

	traceID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{
		"trace_id":  traceID,
		"device_id": deviceID,
	})
	topic := fmt.Sprintf("aivision/edge/%s/cmd/stream_status", deviceID)

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

	var status *ipc.StreamStatusParams
	var jsStatus struct {
		Status  string `json:"status"`
		PlayURL string `json:"play_url"`
	}
	if json.Unmarshal([]byte(respPayload), &jsStatus) == nil && jsStatus.Status != "" {
		status = &ipc.StreamStatusParams{
			Status:  jsStatus.Status,
			PlayURL: jsStatus.PlayURL,
		}
	}
	if status == nil {
		status = ipc.FlatBuffersToStreamStatus([]byte(respPayload))
	}

	if status != nil {
		return StreamStatus{
			DeviceID: deviceID,
			Status:   status.Status,
			PlayURL:  status.PlayURL,
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
	topic := fmt.Sprintf("aivision/edge/self_check/cmd") // Generic topic or node specific if we want

	if c.mqttClient == nil {
		return fmt.Errorf("MQTT client is nil")
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

	var result *ipc.AlgoLoadResultParams
	var jsResult struct {
		Success      bool   `json:"success"`
		ErrorMessage string `json:"error_message"`
		ErrorCode    string `json:"error_code"`
	}
	if json.Unmarshal([]byte(respPayload), &jsResult) == nil {
		result = &ipc.AlgoLoadResultParams{
			Success:      jsResult.Success,
			ErrorMessage: jsResult.ErrorMessage,
			ErrorCode:    jsResult.ErrorCode,
		}
	}
	if result == nil {
		result = ipc.FlatBuffersToAlgoLoadResult([]byte(respPayload))
	}

	if result == nil {
		return fmt.Errorf("MQTT: self check response unparseable")
	}

	if !result.Success {
		return fmt.Errorf("MQTT: algorithm self-check failed: %s (error code: %s)", result.ErrorMessage, result.ErrorCode)
	}

	return nil
}

// UpdateFaceLibrary publishes an UpdateFaceLibrary command via MQTT.
func (c *MqttEngineClient) UpdateFaceLibrary(ctx context.Context, algoName string, faceLibraryJSON []byte) error {
	c.logger.Info("MQTT: UpdateFaceLibrary", zap.String("algo_name", algoName))

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

	topic := "aivision/edge/face_library/cmd"
	if c.mqttClient == nil {
		return fmt.Errorf("MQTT client is nil")
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
func (c *MqttEngineClient) ExtractFaceEmbedding(ctx context.Context, algoName, algoVersion, soPath, algoParamsJSON string, imageBytes []byte) (FaceEmbeddingResult, error) {
	c.logger.Info("MQTT: ExtractFaceEmbedding", zap.String("algo_name", algoName))

	traceID := uuid.New().String()
	payload, err := json.Marshal(struct {
		TraceID        string `json:"trace_id"`
		AlgoName       string `json:"algo_name"`
		AlgoVersion    string `json:"algo_version,omitempty"`
		SoPath         string `json:"so_path,omitempty"`
		AlgoParamsJSON string `json:"algo_params_json,omitempty"`
		ImageBytes     []byte `json:"image_bytes"`
	}{
		TraceID:        traceID,
		AlgoName:       algoName,
		AlgoVersion:    algoVersion,
		SoPath:         soPath,
		AlgoParamsJSON: algoParamsJSON,
		ImageBytes:     imageBytes,
	})
	if err != nil {
		return FaceEmbeddingResult{}, err
	}

	topic := "aivision/edge/face_embedding/cmd"
	if c.mqttClient == nil {
		return FaceEmbeddingResult{}, fmt.Errorf("MQTT client is nil")
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
