package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/pkg/ipc"
)

// maxIPCRespSize 单次 IPC 响应最大字节数（64MB），防止异常引擎返回超大长度导致 OOM
const maxIPCRespSize = 64 * 1024 * 1024

// IPCEngineClient 通过 FlatBuffers + TCP 与 C++ 推理引擎通信
type IPCEngineClient struct {
	addr    string
	timeout time.Duration
	logger  *zap.Logger
}

// NewIPCEngineClient 创建 IPC 引擎客户端
func NewIPCEngineClient(addr string, timeout time.Duration, logger *zap.Logger) *IPCEngineClient {
	return &IPCEngineClient{
		addr:    addr,
		timeout: timeout,
		logger:  logger,
	}
}

func (c *IPCEngineClient) sendCommand(ctx context.Context, cmdType uint32, payload []byte) ([]byte, error) {
	return c.sendCommandWithTimeout(ctx, cmdType, payload, c.timeout)
}

// sendCommandWithTimeout 发送 IPC 命令并等待响应，使用自定义超时（用于长时间操作如自检）。
func (c *IPCEngineClient) sendCommandWithTimeout(_ context.Context, cmdType uint32, payload []byte, timeout time.Duration) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", c.addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("dial tcp: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	// 信封格式：[4字节命令类型][4字节payload长度][payload]
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], cmdType)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(payload)))

	if _, err := conn.Write(header); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}
	if len(payload) > 0 {
		if _, err := conn.Write(payload); err != nil {
			return nil, fmt.Errorf("write payload: %w", err)
		}
	}

	// 读取响应：[4字节响应码][4字节payload长度][payload]
	respHeader := make([]byte, 8)
	if _, err := io.ReadFull(conn, respHeader); err != nil {
		return nil, fmt.Errorf("read resp header: %w", err)
	}
	respLen := binary.LittleEndian.Uint32(respHeader[4:8])
	if respLen > maxIPCRespSize {
		return nil, fmt.Errorf("response too large: %d bytes (max %d)", respLen, maxIPCRespSize)
	}
	respPayload := make([]byte, respLen)
	if respLen > 0 {
		if _, err := io.ReadFull(conn, respPayload); err != nil {
			return nil, fmt.Errorf("read resp payload: %w", err)
		}
	}

	return respPayload, nil
}

// StartStream 发送 StreamStart 指令给 C++ 引擎
func (c *IPCEngineClient) StartStream(ctx context.Context, req StreamStartRequest) (StreamInfo, error) {
	c.logger.Info("IPC: StartStream", zap.String("device_id", req.DeviceID))

	// 构建 FlatBuffers payload
	params := &ipc.StartStreamParams{
		DeviceID:       req.DeviceID,
		StreamURL:      req.RtspURL,
		DecodeHWType:   0, // Auto
		EnableInfer:    req.EnableInfer,
		EnablePlayback: req.EnablePlayback,
		AlgoName:       req.AlgoName,
		AlgoVersion:    req.AlgoVersion,
		SoPath:         req.SoPath,
		AlgoParamsJSON: req.AlgoParamsJSON,
	}
	fbData := ipc.StartStreamParamsToFlatBuffers(params)

	// 201 = StreamStart
	resp, err := c.sendCommand(ctx, 201, fbData)
	if err != nil {
		return StreamInfo{}, err
	}

	// 解析返回的 StreamInfo
	status := ipc.FlatBuffersToStreamStatus(resp)
	if status == nil {
		return StreamInfo{}, fmt.Errorf("invalid stream status response")
	}
	if status.Status != "running" {
		return StreamInfo{}, fmt.Errorf("engine failed to start stream")
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

// StopStream 发送 StreamStop 指令
func (c *IPCEngineClient) StopStream(ctx context.Context, deviceID string) error {
	c.logger.Info("IPC: StopStream", zap.String("device_id", deviceID))

	// 202 = StreamStop
	payload := []byte(deviceID)
	_, err := c.sendCommand(ctx, 202, payload)
	return err
}

// StartPlayback 发送 StreamPlaybackStart 指令
func (c *IPCEngineClient) StartPlayback(ctx context.Context, req StreamStartRequest) (string, error) {
	c.logger.Info("IPC: StartPlayback", zap.String("device_id", req.DeviceID))

	params := &ipc.StartStreamParams{
		DeviceID:       req.DeviceID,
		StreamURL:      req.RtspURL,
		DecodeHWType:   0,
		EnablePlayback: true,
	}
	payload := ipc.StartStreamParamsToFlatBuffers(params)
	resp, err := c.sendCommand(ctx, 203, payload)
	if err != nil {
		return "", err
	}

	status := ipc.FlatBuffersToStreamStatus(resp)
	if status == nil {
		return "", fmt.Errorf("invalid playback status response")
	}
	if status.Status != "running" {
		return "", fmt.Errorf("engine failed to start playback")
	}
	if status.PlayURL != "" {
		return status.PlayURL, nil
	}
	return fmt.Sprintf("rtsp://engine:554/live/%s", req.DeviceID), nil
}

// StopPlayback 发送 StreamPlaybackStop 指令
func (c *IPCEngineClient) StopPlayback(ctx context.Context, deviceID string) error {
	c.logger.Info("IPC: StopPlayback", zap.String("device_id", deviceID))

	// 204 = StreamPlaybackStop
	payload := []byte(deviceID)
	_, err := c.sendCommand(ctx, 204, payload)
	return err
}

// GetStreamStatus 发送 StreamStatus 查询
func (c *IPCEngineClient) GetStreamStatus(ctx context.Context, deviceID string) (StreamStatus, error) {
	c.logger.Debug("IPC: GetStreamStatus", zap.String("device_id", deviceID))

	// 205 = StreamStatus 查询
	payload := []byte(deviceID)
	resp, err := c.sendCommand(ctx, 205, payload)
	if err != nil {
		return StreamStatus{}, err
	}

	status := ipc.FlatBuffersToStreamStatus(resp)
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

// StartSelfCheck 发送 StartSelfCheck 指令给 C++ 引擎，并同步等待返回自检结果
func (c *IPCEngineClient) StartSelfCheck(ctx context.Context, downloadURL, token, algoName, version string) error {
	c.logger.Info("IPC: StartSelfCheck", zap.String("algo_name", algoName), zap.String("version", version))

	// 构建 FlatBuffers payload
	fbData := ipc.StartSelfCheckCmdToFlatBuffers(downloadURL, token, algoName, version)

	// 206 = StartSelfCheck — 使用较长超时，引擎需要下载+解压+dlopen+自检
	resp, err := c.sendCommandWithTimeout(ctx, 206, fbData, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("发送自检命令失败: %w", err)
	}

	result := ipc.FlatBuffersToAlgoLoadResult(resp)
	if result == nil {
		return fmt.Errorf("引擎未返回自检结果")
	}

	if !result.Success {
		return fmt.Errorf("算法自检失败: %s (错误码: %s)", result.ErrorMessage, result.ErrorCode)
	}

	return nil
}
