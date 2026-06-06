package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	streamManager   *StreamManager
	zlmBaseURL      string
	zlmSecret       string
}

// NewMediaService creates a new MediaService.
func NewMediaService(
	zlmClient *zlm.Client,
	mediaStreamRepo *repository.MediaStreamRepository,
	deviceRepo *repository.DeviceRepository,
	streamManager *StreamManager,
	zlmBaseURL string,
	zlmSecret string,
) *MediaService {
	return &MediaService{
		zlmClient:       zlmClient,
		mediaStreamRepo: mediaStreamRepo,
		deviceRepo:      deviceRepo,
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
func (s *MediaService) GetPlayURL(ctx context.Context, deviceID, protocol, streamType string) (string, error) {
	// 1. 通过 StreamManager 获取流引用
	err := s.streamManager.Acquire(ctx, deviceID, "play", map[string]string{
		"protocol":    protocol,
		"stream_type": streamType,
	})
	if err != nil {
		return "", fmt.Errorf("acquire stream: %w", err)
	}

	// 2. 获取流状态以拿到播放地址
	state := s.streamManager.GetStream(ctx, deviceID)
	if state == nil {
		return "", errors.New(errors.ErrInternal, "stream state not found after acquire")
	}

	app := "live"
	stream := deviceID
	if streamType == "sub" || streamType == "auxiliary" {
		stream = deviceID + "_sub"
	}

	// Generate signed token
	token := s.GeneratePlayToken(stream, "anonymous", 30*time.Minute)

	switch protocol {
	case "flv":
		return fmt.Sprintf("http://%s:80/%s/%s.flv?token=%s", s.zlmBaseURL, app, stream, token), nil
	case "hls":
		return fmt.Sprintf("http://%s:80/%s/%s/hls.m3u8?token=%s", s.zlmBaseURL, app, stream, token), nil
	default: // auto, webrtc
		return fmt.Sprintf("webrtc://%s:8000/%s/%s?token=%s", s.zlmBaseURL, app, stream, token), nil
	}
}

// StopPlayURL closes the stream proxy for a device (stop preview).
func (s *MediaService) StopPlayURL(ctx context.Context, deviceID string) error {
	return s.streamManager.Release(ctx, deviceID, "play")
}

// GetSnapshot captures a snapshot from the device stream.
func (s *MediaService) GetSnapshot(ctx context.Context, deviceID string) ([]byte, error) {
	device, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return nil, errors.New(errors.ErrNotFound, "device not found")
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

