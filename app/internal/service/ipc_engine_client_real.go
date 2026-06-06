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

// IPCEngineClient 通过 FlatBuffers + UDS 与 C++ 推理引擎通信
type IPCEngineClient struct {
	socketPath string
	timeout    time.Duration
	logger     *zap.Logger
}

// NewIPCEngineClient 创建 IPC 引擎客户端
func NewIPCEngineClient(socketPath string, timeout time.Duration, logger *zap.Logger) *IPCEngineClient {
	return &IPCEngineClient{
		socketPath: socketPath,
		timeout:    timeout,
		logger:     logger,
	}
}

func (c *IPCEngineClient) sendCommand(ctx context.Context, cmdType uint32, payload []byte) ([]byte, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("dial uds: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(c.timeout))

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
		DeviceID:    req.DeviceID,
		StreamURL:   req.RtspURL,
		DecodeHWType: 0, // Auto
	}
	fbData := ipc.StartStreamParamsToFlatBuffers(params)

	// 201 = StreamStart
	resp, err := c.sendCommand(ctx, 201, fbData)
	if err != nil {
		return StreamInfo{}, err
	}

	// 解析返回的 StreamInfo
	status := ipc.FlatBuffersToStreamStatus(resp)
	if status != nil {
		return StreamInfo{
			DeviceID: req.DeviceID,
			Status:   status.Status,
		}, nil
	}

	return StreamInfo{
		DeviceID: req.DeviceID,
		Status:   "active",
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
func (c *IPCEngineClient) StartPlayback(ctx context.Context, deviceID string) (string, error) {
	c.logger.Info("IPC: StartPlayback", zap.String("device_id", deviceID))

	// 203 = StreamPlaybackStart
	payload := []byte(deviceID)
	_, err := c.sendCommand(ctx, 203, payload)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("rtsp://engine:554/live/%s", deviceID), nil
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
