package service

import (
	"context"
)

// StreamStartRequest 启动流请求
type StreamStartRequest struct {
	DeviceID       string `json:"device_id"`
	RtspURL        string `json:"rtsp_url"`
	EnableInfer    bool   `json:"enable_infer"`
	EnablePlayback bool   `json:"enable_playback"`
	AlgoName       string `json:"algo_name,omitempty"`
	AlgoVersion    string `json:"algo_version,omitempty"`
	SoPath         string `json:"so_path,omitempty"`
	AlgoParamsJSON string `json:"algo_params_json,omitempty"`
}

// StreamInfo 流启动后的信息
type StreamInfo struct {
	DeviceID string `json:"device_id"`
	Status   string `json:"status"`
	PlayURL  string `json:"play_url"` // C++ 引擎上报的推流地址或播放地址
}

// StreamStatus 流状态信息
type StreamStatus struct {
	DeviceID   string `json:"device_id"`
	Status     string `json:"status"`
	PlayURL    string `json:"play_url"`
	RetryCount int    `json:"retry_count"`
}

// FaceEmbeddingResult 是 Engine 从单张人脸图库图片提取出的特征。
type FaceEmbeddingResult struct {
	Success        bool      `json:"success"`
	ErrorCode      string    `json:"error_code,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	Embedding      []float32 `json:"embedding,omitempty"`
	FaceConfidence float64   `json:"face_confidence,omitempty"`
	QualityScore   float64   `json:"quality_score,omitempty"`
	EmbeddingNorm  float64   `json:"embedding_norm,omitempty"`
}

// EngineClient 抽象 Go 控制面与 C++ 推理引擎的通信能力。
// 当前生产实现为 MqttEngineClient：控制命令经 MQTT 下发，响应通过 trace_id
// 与 Redis Pub/Sub 配对；MockEngineClient 仅用于测试和本地降级。
type EngineClient interface {
	StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error)
	StopStream(ctx context.Context, deviceID string) error
	StartPlayback(ctx context.Context, req StreamStartRequest) (string, error)
	StopPlayback(ctx context.Context, deviceID string) error
	GetStreamStatus(ctx context.Context, deviceID string) (StreamStatus, error)
	StartSelfCheck(ctx context.Context, downloadURL, token, algoName, version string) error
	WarmupAlgorithm(ctx context.Context, nodeID, algoName, algoVersion string) error
	UpdateFaceLibrary(ctx context.Context, nodeID, algoName string, faceLibraryJSON []byte) error
	ExtractFaceEmbedding(ctx context.Context, nodeID, algoName, algoVersion string, imageBytes []byte) (FaceEmbeddingResult, error)
}

// MockEngineClient 模拟实现
type MockEngineClient struct{}

func (m *MockEngineClient) StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error) {
	return StreamInfo{
		DeviceID: req.DeviceID,
		Status:   "active",
		PlayURL:  "rtsp://mock-engine:554/live/" + req.DeviceID,
	}, nil
}

func (m *MockEngineClient) StopStream(ctx context.Context, deviceID string) error {
	return nil
}

func (m *MockEngineClient) StartPlayback(ctx context.Context, req StreamStartRequest) (string, error) {
	return "rtsp://mock-engine:554/live/" + req.DeviceID, nil
}

func (m *MockEngineClient) StopPlayback(ctx context.Context, deviceID string) error {
	return nil
}

func (m *MockEngineClient) GetStreamStatus(ctx context.Context, deviceID string) (StreamStatus, error) {
	return StreamStatus{
		DeviceID: deviceID,
		Status:   "active",
		PlayURL:  "rtsp://mock-engine:554/live/" + deviceID,
	}, nil
}

func (m *MockEngineClient) StartSelfCheck(ctx context.Context, downloadURL, token, algoName, version string) error {
	return nil
}

func (m *MockEngineClient) WarmupAlgorithm(ctx context.Context, nodeID, algoName, algoVersion string) error {
	return nil
}

func (m *MockEngineClient) UpdateFaceLibrary(ctx context.Context, nodeID, algoName string, faceLibraryJSON []byte) error {
	return nil
}

func (m *MockEngineClient) ExtractFaceEmbedding(ctx context.Context, nodeID, algoName, algoVersion string, imageBytes []byte) (FaceEmbeddingResult, error) {
	return FaceEmbeddingResult{Success: true, Embedding: make([]float32, 512)}, nil
}
