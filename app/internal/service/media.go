package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// MediaService handles media streaming orchestration.
type MediaService struct {
	zlmClient       *zlm.Client
	mediaStreamRepo *repository.MediaStreamRepository
	deviceRepo      *repository.DeviceRepository
	edgeNodeRepo    *repository.EdgeNodeRepository
	streamManager   *StreamManager
	zlmBaseURL      string
	zlmSecret       string
}

// NewMediaService creates a new MediaService.
func NewMediaService(
	zlmClient *zlm.Client,
	mediaStreamRepo *repository.MediaStreamRepository,
	deviceRepo *repository.DeviceRepository,
	edgeNodeRepo *repository.EdgeNodeRepository,
	streamManager *StreamManager,
	zlmBaseURL string,
	zlmSecret string,
) *MediaService {
	return &MediaService{
		zlmClient:       zlmClient,
		mediaStreamRepo: mediaStreamRepo,
		deviceRepo:      deviceRepo,
		edgeNodeRepo:    edgeNodeRepo,
		streamManager:   streamManager,
		zlmBaseURL:      zlmBaseURL,
		zlmSecret:       zlmSecret,
	}
}

// GeneratePlayToken generates a signed playback token.
func (s *MediaService) GeneratePlayToken(streamID string, userID string, expiry time.Duration) string {
	expires := time.Now().Add(expiry).Unix()
	mac := hmac.New(sha256.New, []byte(s.zlmSecret))
	data := fmt.Sprintf("%s:%s:%d", streamID, userID, expires)
	mac.Write([]byte(data))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s_%d_%s", userID, expires, sig)
}

// VerifyPlayAuth verifies the playback token from ZLM on_play webhook.
// params format: "token=xxx&user_id=xxx"
func (s *MediaService) VerifyPlayAuth(ctx context.Context, app, stream, params string) error {
	// TODO: 解析 params 中的 token，验证 HMAC 签名和过期时间
	return nil
}

// GetPlayURL generates a playback URL for the given device.
// Supported protocols: auto, webrtc, flv, hls.
// streamType: "main" for main stream, "sub" for sub-stream.
// For RTSP devices, this will automatically start pulling the stream via StreamManager.
func (s *MediaService) GetPlayURL(ctx context.Context, deviceID, protocol, streamType string) (string, string, error) {
	app := "live"
	stream := playbackStreamID(deviceID, streamType)

	metadata := map[string]string{
		"protocol":    protocol,
		"stream_type": streamType,
	}

	// 1. 动态节点选择：优先使用已有流关联节点，次选任意在线边缘节点
	if state := s.streamManager.GetStream(ctx, stream); state != nil && state.NodeID != "" {
		metadata["target_node_id"] = state.NodeID
	} else if s.edgeNodeRepo != nil {
		if node, err := s.edgeNodeRepo.FindAnyOnlineNode(ctx); err == nil && node != nil {
			metadata["target_node_id"] = node.ID
		}
	}

	// 2. 通过 StreamManager 获取流引用 (使用 stream 标识主流与子流)
	err := s.streamManager.Acquire(ctx, stream, "play", metadata)
	if err != nil {
		return "", "", fmt.Errorf("acquire stream: %w", err)
	}

	// 3. 获取流状态以拿到播放地址
	state := s.streamManager.GetStream(ctx, stream)
	if state == nil {
		_ = s.streamManager.Release(ctx, stream, "play")
		return "", "", errors.New(errors.ErrStreamStateNotFound, "")
	}

	zlmHost := state.ZLMHost
	// 如果边缘节点上报的 zlmHost 是环回地址（localhost/127.0.0.1），从关联边缘节点的 Endpoint 替换为实际节点 IP
	if (zlmHost == "" || zlmHost == "localhost" || zlmHost == "127.0.0.1" || zlmHost == "::1") && state.NodeID != "" && s.edgeNodeRepo != nil {
		if node, err := s.edgeNodeRepo.FindByID(ctx, state.NodeID); err == nil && node != nil && node.Endpoint != "" {
			if u, err := url.Parse(node.Endpoint); err == nil && u.Hostname() != "" {
				zlmHost = u.Hostname()
			} else {
				h := strings.TrimPrefix(strings.TrimPrefix(node.Endpoint, "http://"), "https://")
				if idx := strings.Index(h, ":"); idx > 0 {
					h = h[:idx]
				}
				if h != "" {
					zlmHost = h
				}
			}
		}
	}

	codec := state.Codec
	if codec == "" {
		codec = "h264"
	}

	return s.buildPlayURL(app, stream, zlmHost, state.ZLMHTTPPort, protocol), codec, nil
}

func (s *MediaService) buildPlayURL(app, stream string, zlmHost string, zlmHTTPPort int, protocol string) string {
	token := s.GeneratePlayToken(stream, "anonymous", 30*time.Minute)

	host := zlmHost
	httpPort := zlmHTTPPort
	if host == "" || httpPort == 0 || httpPort == 80 {
		if u, err := url.Parse(s.zlmBaseURL); err == nil {
			if host == "" && u.Hostname() != "" {
				host = u.Hostname()
			}
			if (httpPort == 0 || httpPort == 80) && u.Port() != "" {
				if p, err := strconv.Atoi(u.Port()); err == nil && p > 0 {
					httpPort = p
				}
			}
		}
		if host == "" {
			host = s.zlmBaseURL
			host = strings.TrimPrefix(host, "http://")
			host = strings.TrimPrefix(host, "https://")
			if idx := strings.Index(host, ":"); idx > 0 {
				host = host[:idx]
			}
		}
		if httpPort == 0 {
			httpPort = 8000
		}
	}

	if protocol == zlm.ProtocolHLS {
		return fmt.Sprintf("http://%s:%d/%s/%s/hls.m3u8?token=%s", host, httpPort, app, stream, token)
	}
	if protocol == zlm.ProtocolFLV {
		return fmt.Sprintf("http://%s:%d/%s/%s.flv?token=%s", host, httpPort, app, stream, token)
	}

	// 默认返回 webrtc:// 协议，供前端 VideoPlayer 自动尝试 WebRTC -> HLS -> FLV 降级
	return fmt.Sprintf("%s://%s:%d/%s/%s?token=%s", zlm.ProtocolWebRTC, host, httpPort, app, stream, token)
}

// StopPlayURL closes the stream proxy for a device and stream type (stop preview).
func (s *MediaService) StopPlayURL(ctx context.Context, deviceID, streamType string) error {
	return s.streamManager.Release(ctx, playbackStreamID(deviceID, streamType), "play")
}

// GetSnapshot captures a snapshot from the device stream.
func (s *MediaService) GetSnapshot(ctx context.Context, deviceID string) ([]byte, error) {
	device, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, errors.New(errors.ErrDeviceNotFound, "")
	}

	url := ""
	if device.AccessType == model.DeviceAccessTypeRTSP {
		url = device.RtspURL
	} else {
		// For GB28181, use the ZLM stream URL
		url = fmt.Sprintf("rtsp://%s:554/live/%s", s.zlmBaseURL, deviceID)
	}

	imgData, err := s.zlmClient.GetSnap(ctx, zlm.GetSnapRequest{
		URL:        url,
		TimeoutSec: 10,
		ExpireSec:  30,
	})
	if err != nil {
		return nil, fmt.Errorf("zlm snapshot: %w", err)
	}

	return imgData, nil
}

// ListStreams returns all active stream states managed by StreamManager.
func (s *MediaService) ListStreams(ctx context.Context) []*StreamState {
	return s.streamManager.ListStreams(ctx)
}
