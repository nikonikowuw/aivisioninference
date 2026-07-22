package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"

	pkgcache "github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/repository"

	"github.com/niko-admin/niko-admin/internal/model"
)

// SIPService handles GB28181 SIP signaling logic.
type SIPService struct {
	deviceRepo          *repository.DeviceRepository
	gbDeviceRepo        *repository.GB28181DeviceRepository
	mediaStreamRepo     *repository.MediaStreamRepository
	discoverySvc        *DeviceDiscoveryService
	smartRecordRepo     *repository.SmartRecordRepository
	taskClient          taskClient
	deviceSipConfigRepo *repository.DeviceSipConfigRepository
	deviceRepoV2        *repository.DeviceRepositoryV2
	auditRepo           *repository.AuditRepository
	streamSessionRepo   *repository.GB28181StreamSessionRepository

	// GB28181 ZLM 集成字段
	zlmClient     *zlm.Client
	streamManager *StreamManager
	zlmBaseIP     string // ZLM 对设备可见的 IP
	rtmpPort      int
	rtspPort      int
	httpPort      int

	// 缓存和 WebSocket
	cache pkgcache.Cache
	hub   *ws.Hub

	// 心跳存活性追踪（Redis / 内存）
	heartbeats HeartbeatStore

	// SIP Runtime
	runtimeSvc *SIPRuntimeService
}

// NewSIPServiceWithZLM creates a SIPService with ZLM media integration.
func NewSIPServiceWithZLM(
	deviceRepo *repository.DeviceRepository,
	gbDeviceRepo *repository.GB28181DeviceRepository,
	mediaStreamRepo *repository.MediaStreamRepository,
	smartRecordRepo *repository.SmartRecordRepository,
	deviceSipConfigRepo *repository.DeviceSipConfigRepository,
	deviceRepoV2 *repository.DeviceRepositoryV2,
	taskClient taskClient,
	zlmClient *zlm.Client,
	streamManager *StreamManager,
	zlmBaseIP string,
	rtmpPort, rtspPort, httpPort int,
	cache pkgcache.Cache,
	hub *ws.Hub,
	auditRepo *repository.AuditRepository,
	streamSessionRepo *repository.GB28181StreamSessionRepository,
	heartbeats HeartbeatStore,
) *SIPService {
	return &SIPService{
		deviceRepo:          deviceRepo,
		gbDeviceRepo:        gbDeviceRepo,
		mediaStreamRepo:     mediaStreamRepo,
		smartRecordRepo:     smartRecordRepo,
		deviceSipConfigRepo: deviceSipConfigRepo,
		deviceRepoV2:        deviceRepoV2,
		taskClient:          taskClient,
		zlmClient:           zlmClient,
		streamManager:       streamManager,
		zlmBaseIP:           zlmBaseIP,
		rtmpPort:            rtmpPort,
		rtspPort:            rtspPort,
		httpPort:            httpPort,
		cache:               cache,
		hub:                 hub,
		auditRepo:           auditRepo,
		streamSessionRepo:   streamSessionRepo,
		heartbeats:          heartbeats,
	}
}

func (s *SIPService) SetDiscoveryService(svc *DeviceDiscoveryService) {
	s.discoverySvc = svc
}

func (s *SIPService) SetRuntimeService(svc *SIPRuntimeService) {
	s.runtimeSvc = svc
}

var gb28181DeviceIDRegex = regexp.MustCompile(`^\d{20}$`)

func (s *SIPService) ValidateDeviceCode(deviceID string) error {
	if !gb28181DeviceIDRegex.MatchString(deviceID) {
		return errors.New(errors.ErrInvalidGB28181DeviceCode, "")
	}
	return nil
}

// HandleRegister validates a device SIP registration request and updates device status.
func (s *SIPService) HandleRegister(ctx context.Context, deviceID, remoteIP string, port int) error {
	if err := s.ValidateDeviceCode(deviceID); err != nil {
		return err
	}

	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return errors.New(errors.ErrDeviceNotFound, "")
	}

	oldIP := gbDevice.RegisterAddress
	oldPort := gbDevice.RegisterPort
	addressChanged := oldIP != "" && (oldIP != remoteIP || oldPort != port)

	now := time.Now()
	updates := map[string]interface{}{
		"register_address": remoteIP,
		"register_port":    port,
		"last_register_at": now,
		"status":           model.GB28181StatusOnline,
	}

	if err := s.gbDeviceRepo.Update(ctx, gbDevice.ID, updates); err != nil {
		return fmt.Errorf("update device registration: %w", err)
	}

	if s.deviceSipConfigRepo != nil {
		if err := s.deviceSipConfigRepo.UpdateRegisterAddress(ctx, deviceID, remoteIP, port); err != nil {
			zap.L().Warn("update DeviceSipConfig register address failed", zap.String("device_code", deviceID), zap.Error(err))
		}
	}

	// Sync Device model status with fallback ExternalKey lookup
	if gbDevice.DeviceID != nil {
		if err := s.deviceRepo.UpdateStatus(ctx, *gbDevice.DeviceID, model.DeviceStatusOnline, "", ""); err != nil {
			zap.L().Warn("update device status failed", zap.String("device_id", *gbDevice.DeviceID), zap.Error(err))
		}
	} else {
		// Fallback: look up device by ExternalKey if no direct association
		key := "gb28181_nvr:" + deviceID
		if dev, err := s.deviceRepo.FindByExternalKey(ctx, key); err == nil && dev != nil {
			if err := s.deviceRepo.UpdateStatus(ctx, dev.ID, model.DeviceStatusOnline, "", ""); err != nil {
				zap.L().Warn("update device status by external key failed", zap.String("key", key), zap.Error(err))
			}
		}
	}

	if addressChanged && s.auditRepo != nil {
		auditLog := &model.AuditLog{
			Username:      "system",
			ActionType:    "update_register_address",
			ResourceType:  "gb28181_device",
			ResourceID:    gbDevice.DeviceCode,
			RequestMethod: "REGISTER",
			RequestIP:     remoteIP,
			ResultSummary: fmt.Sprintf("Device %s address changed from %s:%d to %s:%d", deviceID, oldIP, oldPort, remoteIP, port),
		}
		if err := s.auditRepo.Create(ctx, auditLog); err != nil {
			zap.L().Warn("audit log create failed", zap.Error(err))
		}
	}

	go func() {
		if err := s.QueryCatalog(context.Background(), gbDevice.DeviceCode); err != nil {
			zap.L().Warn("catalog query failed", zap.String("device_code", gbDevice.DeviceCode), zap.Error(err))
		}
	}()

	s.broadcastStatus(deviceID, model.DeviceStatusOnline)
	return nil
}

// HandleInviteOK marks an INVITE session as active after receiving 200 OK from the device.
func (s *SIPService) HandleInviteOK(ctx context.Context, callID string, sdpBody string) error {
	session, err := s.streamSessionRepo.FindByCallID(ctx, callID)
	if err != nil {
		return fmt.Errorf("find session by call id: %w", err)
	}
	updates := map[string]interface{}{
		"status":     "active",
		"start_time": time.Now(),
	}
	return s.streamSessionRepo.Update(ctx, session.StreamID, updates)
}

// HandleHeartbeat records the heartbeat in store and DB, and transitions device to online if needed.
func (s *SIPService) HandleHeartbeat(ctx context.Context, gbDevice *model.GB28181Device) error {
	deviceID := gbDevice.DeviceCode

	// Record heartbeat in store (fast, no DB).
	if err := s.heartbeats.Record(ctx, deviceID, time.Now()); err != nil {
		zap.L().Warn("record heartbeat in store failed", zap.String("device_code", deviceID), zap.Error(err))
	}

	// Sync heartbeat timestamp to DeviceSipConfig
	if s.deviceSipConfigRepo != nil {
		if err := s.deviceSipConfigRepo.UpdateHeartbeat(ctx, deviceID); err != nil {
			zap.L().Debug("update DeviceSipConfig heartbeat failed", zap.String("device_code", deviceID), zap.Error(err))
		}
	}

	// Sync last_heartbeat_at on gb28181_devices for API display.
	if err := s.gbDeviceRepo.Update(ctx, gbDevice.ID, map[string]interface{}{
		"last_heartbeat_at": time.Now(),
	}); err != nil {
		zap.L().Warn("update last_heartbeat_at failed", zap.String("device_code", deviceID), zap.Error(err))
	}

	if gbDevice.Status != model.GB28181StatusOnline {
		if err := s.gbDeviceRepo.UpdateStatus(ctx, gbDevice.ID, model.GB28181StatusOnline); err != nil {
			zap.L().Warn("update device status to online failed", zap.String("device_code", deviceID), zap.Error(err))
		}
		s.broadcastStatus(deviceID, model.DeviceStatusOnline)
	}
	return nil
}

// CheckHeartbeatTimeout checks heartbeat store for expired devices and marks them offline.
func (s *SIPService) CheckHeartbeatTimeout(ctx context.Context, timeout time.Duration) ([]model.GB28181Device, error) {
	cutoff := time.Now().Add(-timeout)
	expiredCodes, err := s.heartbeats.GetExpired(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("get expired heartbeats: %w", err)
	}
	if len(expiredCodes) == 0 {
		return nil, nil
	}

	// Batch load devices from DB to verify they still exist and are online.
	devices, err := s.gbDeviceRepo.FindByDeviceCodes(ctx, expiredCodes)
	if err != nil {
		return nil, fmt.Errorf("find devices by codes: %w", err)
	}

	deviceMap := make(map[string]model.GB28181Device, len(devices))
	for _, dev := range devices {
		deviceMap[dev.DeviceCode] = dev
	}

	var offlineDevices []model.GB28181Device
	var firstErr error
	for _, code := range expiredCodes {
		dev, ok := deviceMap[code]
		if !ok || dev.Status != model.GB28181StatusOnline {
			// Device doesn't exist or is already offline — clean up stale store entry.
			if err := s.heartbeats.Remove(ctx, code); err != nil {
				zap.L().Warn("failed to remove stale heartbeat entry from store",
					zap.String("device_code", code), zap.Error(err))
			}
			continue
		}

		if err := s.gbDeviceRepo.UpdateStatus(ctx, dev.ID, model.GB28181StatusOffline); err != nil {
			zap.L().Warn("update device to offline failed", zap.String("device_code", dev.DeviceCode), zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if dev.DeviceID != nil {
			if err := s.deviceRepo.UpdateStatus(ctx, *dev.DeviceID, model.DeviceStatusOffline, "", ""); err != nil {
				zap.L().Warn("update linked device to offline failed", zap.String("device_id", *dev.DeviceID), zap.Error(err))
			}
		}
		s.broadcastStatus(dev.DeviceCode, model.DeviceStatusOffline)

		// Remove from store after successful offline transition.
		if err := s.heartbeats.Remove(ctx, code); err != nil {
			zap.L().Warn("failed to remove device from heartbeat store",
				zap.String("device_code", code), zap.Error(err))
		}

		offlineDevices = append(offlineDevices, dev)
	}

	return offlineDevices, firstErr
}

func (s *SIPService) broadcastStatus(deviceCode, status string) {
	if s.hub == nil {
		return
	}
	msg := &ws.Message{
		Type: "device_status_change",
		Payload: map[string]interface{}{
			"deviceCode": deviceCode,
			"status":     status,
			"time":       time.Now().Unix(),
		},
	}
	s.hub.Broadcast(msg)
}

// BuildCatalogueResponse generates a GB28181 MANSCDP XML catalogue response.
func (s *SIPService) BuildCatalogueResponse(ctx context.Context, deviceID, sn string) (string, error) {
	if err := s.ValidateDeviceCode(deviceID); err != nil {
		return "", err
	}
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return "", err
	}
	status := "ON"
	if gbDevice.Status == model.GB28181StatusOffline {
		status = "OFF"
	}
	escapedSN := html.EscapeString(sn)
	escapedDeviceID := html.EscapeString(deviceID)
	escapedName := html.EscapeString(gbDevice.DeviceCode)
	escapedManufacturer := html.EscapeString(gbDevice.Manufacturer)
	escapedModel := html.EscapeString(gbDevice.Model)
	escapedStatus := html.EscapeString(status)

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
<CmdType>Catalog</CmdType>
<SN>%s</SN>
<DeviceID>%s</DeviceID>
<SumNum>1</SumNum>
<DeviceList>
<Item>
<DeviceID>%s</DeviceID>
<Name>%s</Name>
<Manufacturer>%s</Manufacturer>
<Model>%s</Model>
<Status>%s</Status>
<Longitude>%.6f</Longitude>
<Latitude>%.6f</Latitude>
</Item>
</DeviceList>
</Response>`, escapedSN, escapedDeviceID, escapedDeviceID, escapedName, escapedManufacturer, escapedModel, escapedStatus, 0.0, 0.0), nil
}

// HandleUnregister processes a device-initiated unregistration (Expires=0).
func (s *SIPService) HandleUnregister(ctx context.Context, deviceID string) error {
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return err
	}
	if err := s.gbDeviceRepo.UpdateStatus(ctx, gbDevice.ID, model.GB28181StatusOffline); err != nil {
		zap.L().Warn("update device to offline on unregister failed", zap.String("device_code", deviceID), zap.Error(err))
	}
	if gbDevice.DeviceID != nil {
		if err := s.deviceRepo.UpdateStatus(ctx, *gbDevice.DeviceID, model.DeviceStatusOffline, "", ""); err != nil {
			zap.L().Warn("update linked device status on unregister failed", zap.String("device_id", *gbDevice.DeviceID), zap.Error(err))
		}
	}
	s.broadcastStatus(deviceID, model.DeviceStatusOffline)
	return nil
}

// QueryCatalog sends a catalog query to the device via the SIP runtime.
func (s *SIPService) QueryCatalog(ctx context.Context, deviceCode string) error {
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
	if err != nil {
		return fmt.Errorf("find device for catalog query: %w", err)
	}
	if gbDevice.Status != model.GB28181StatusOnline {
		return errors.New(errors.ErrGB28181DeviceOffline, "")
	}

	if s.runtimeSvc == nil || !s.runtimeSvc.IsRunning() {
		// SIP runtime not available — log and return nil (caller may be using ZLM SIP stack)
		zap.L().Debug("SIP runtime not available for catalog query, skipping",
			zap.String("device_code", deviceCode))
		return nil
	}

	sn, err := s.runtimeSvc.SendCatalogQuery(ctx, deviceCode)
	if err != nil {
		return fmt.Errorf("send catalog query: %w", err)
	}

	if s.cache != nil {
		taskID := uuid.NewString()
		cacheKey := "catalog_task:device:" + deviceCode
		if err := s.cache.Set(ctx, cacheKey, []byte(taskID), 5*time.Minute); err != nil {
			zap.L().Warn("set catalog task cache failed", zap.String("device_code", deviceCode), zap.Error(err))
			return nil // Non-fatal: query was sent, cache is optional
		}
		status := map[string]interface{}{"task_id": taskID, "status": "sent", "sn": sn}
		statusData, _ := json.Marshal(status)
		if err := s.cache.Set(ctx, "catalog_task:"+taskID, statusData, 5*time.Minute); err != nil {
			// Clean up the device key if status key fails
			_ = s.cache.Del(ctx, cacheKey)
			zap.L().Warn("set catalog task status cache failed", zap.String("task_id", taskID), zap.Error(err))
		}
	}
	return nil
}

// StartLiveStream initiates a live stream from a GB28181 device via SIP INVITE + ZLM RTP receive.
func (s *SIPService) StartLiveStream(ctx context.Context, deviceCode, streamID string) (string, error) {
	if s.zlmClient == nil {
		return "", errors.New(errors.ErrZLMNotConfigured, "")
	}
	if s.runtimeSvc == nil || !s.runtimeSvc.IsRunning() {
		return "", errors.New(errors.ErrZLMNotConfigured, "SIP runtime not available")
	}

	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
	if err != nil {
		return "", fmt.Errorf("find device: %w", err)
	}

	port, err := s.zlmClient.OpenRtpServer(ctx, zlm.OpenRtpServerRequest{StreamID: streamID})
	if err != nil {
		return "", fmt.Errorf("open rtp server: %w", err)
	}

	ssrc := buildSSRC(deviceCode, "0")
	callID, err := s.runtimeSvc.SendInvite(ctx, gbDevice.DeviceCode, gbDevice.DeviceCode, port, ssrc)
	if err != nil {
		if closeErr := s.zlmClient.CloseRtpServer(ctx, streamID); closeErr != nil {
			zap.L().Warn("close rtp server after invite failure", zap.Error(closeErr))
		}
		return "", fmt.Errorf("send invite: %w", err)
	}

	now := time.Now()
	session := &model.GB28181StreamSession{
		StreamID: streamID, DeviceCode: deviceCode, ChannelID: deviceCode,
		ZLMRTPPort: port, SIPCallID: callID, SSRC: ssrc, Status: "pending", StartTime: &now,
	}
	if err := s.streamSessionRepo.Create(ctx, session); err != nil {
		zap.L().Warn("create stream session failed", zap.String("stream_id", streamID), zap.Error(err))
	}

	if s.streamManager != nil {
		if err := s.streamManager.Acquire(ctx, deviceCode, "gb28181_live", map[string]string{"zlm_stream_id": streamID, "sip_call_id": callID}); err != nil {
			zap.L().Warn("StreamManager Acquire failed", zap.String("device", deviceCode), zap.String("stream", streamID), zap.Error(err))
		}
	}

	return fmt.Sprintf("webrtc://%s:%d/live/%s", s.zlmBaseIP, s.httpPort, streamID), nil
}

// StopLiveStream stops a live stream by sending SIP BYE and releasing ZLM resources.
func (s *SIPService) StopLiveStream(ctx context.Context, deviceCode, streamID string) error {
	if s.zlmClient == nil {
		return errors.New(errors.ErrZLMNotConfigured, "")
	}

	session, err := s.streamSessionRepo.FindByStreamID(ctx, streamID)
	if err == nil && session != nil && s.runtimeSvc != nil {
		if err := s.runtimeSvc.SendBye(ctx, session.DeviceCode, session.ChannelID, session.SIPCallID); err != nil {
			zap.L().Warn("send BYE failed", zap.String("stream_id", streamID), zap.Error(err))
		}
	} else if err != nil {
		zap.L().Warn("stream session not found for stop", zap.String("stream_id", streamID), zap.Error(err))
	}

	if err := s.zlmClient.CloseRtpServer(ctx, streamID); err != nil {
		zap.L().Warn("close rtp server failed", zap.String("stream_id", streamID), zap.Error(err))
	}

	if s.streamManager != nil {
		if err := s.streamManager.Release(ctx, deviceCode, "gb28181_live"); err != nil {
			zap.L().Warn("StreamManager Release failed", zap.String("device", deviceCode), zap.Error(err))
		}
	}

	if err := s.streamSessionRepo.Delete(ctx, streamID); err != nil {
		zap.L().Warn("delete stream session failed", zap.String("stream_id", streamID), zap.Error(err))
	}
	return nil
}

// StartPlayback initiates a playback stream from a GB28181 device via SIP INVITE.
func (s *SIPService) StartPlayback(ctx context.Context, deviceCode, streamID string, start, end time.Time) (string, error) {
	if s.zlmClient == nil {
		return "", errors.New(errors.ErrZLMNotConfigured, "")
	}
	if s.runtimeSvc == nil || !s.runtimeSvc.IsRunning() {
		return "", errors.New(errors.ErrZLMNotConfigured, "SIP runtime not available")
	}
	if err := s.ValidateDeviceCode(deviceCode); err != nil {
		return "", err
	}
	if end.Before(start) {
		return "", errors.New(errors.ErrTimeRangeOrder, "")
	}

	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
	if err != nil {
		return "", fmt.Errorf("find device: %w", err)
	}

	port, err := s.zlmClient.OpenRtpServer(ctx, zlm.OpenRtpServerRequest{StreamID: streamID})
	if err != nil {
		return "", fmt.Errorf("open rtp server: %w", err)
	}

	ssrc := buildSSRC(deviceCode, "1")
	callID, err := s.runtimeSvc.SendPlaybackInvite(ctx, gbDevice.DeviceCode, gbDevice.DeviceCode, port, ssrc, start, end)
	if err != nil {
		if closeErr := s.zlmClient.CloseRtpServer(ctx, streamID); closeErr != nil {
			zap.L().Warn("close rtp server after playback invite failure", zap.Error(closeErr))
		}
		return "", fmt.Errorf("send playback invite: %w", err)
	}

	now := time.Now()
	session := &model.GB28181StreamSession{
		StreamID: streamID, DeviceCode: deviceCode, ChannelID: deviceCode, StreamType: "playback",
		ZLMRTPPort: port, SIPCallID: callID, SSRC: ssrc, Status: "pending", StartTime: &now,
	}
	if err := s.streamSessionRepo.Create(ctx, session); err != nil {
		zap.L().Warn("create playback stream session failed", zap.String("stream_id", streamID), zap.Error(err))
	}

	// Register with StreamManager with rollback on failure
	if s.streamManager != nil {
		if err := s.streamManager.Acquire(ctx, deviceCode, "gb28181_playback", map[string]string{"zlm_stream_id": streamID, "sip_call_id": callID}); err != nil {
			// Rollback ZLM resources
			_ = s.zlmClient.CloseRtpServer(ctx, streamID)
			if session.SIPCallID != "" && s.runtimeSvc != nil {
				_ = s.runtimeSvc.SendBye(ctx, session.DeviceCode, session.ChannelID, session.SIPCallID)
			}
			zap.L().Warn("StreamManager Acquire failed, rolled back ZLM resources",
				zap.String("device", deviceCode), zap.String("stream", streamID), zap.Error(err))
			return "", fmt.Errorf("stream manager acquire: %w", err)
		}
	}

	return fmt.Sprintf("http://%s:%d/live/%s/hls.m3u8", s.zlmBaseIP, s.httpPort, streamID), nil
}

// PlaybackControl sends playback control commands (pause/play/seek) to the device.
func (s *SIPService) PlaybackControl(ctx context.Context, streamID, action string, speed float64, stamp int64) error {
	if s.runtimeSvc == nil || !s.runtimeSvc.IsRunning() {
		return errors.New(errors.ErrZLMNotConfigured, "SIP runtime not available")
	}
	session, err := s.streamSessionRepo.FindByStreamID(ctx, streamID)
	if err != nil {
		return fmt.Errorf("find stream session: %w", err)
	}
	return s.runtimeSvc.SendPlaybackControl(ctx, session.DeviceCode, session.ChannelID, session.SIPCallID, action, speed, stamp)
}

// StopPlayback stops a playback stream.
func (s *SIPService) StopPlayback(ctx context.Context, deviceCode, streamID string) error {
	return s.StopLiveStream(ctx, deviceCode, streamID)
}

// SyncCatalogChannels synchronizes catalog channel responses for an NVR device.
func (s *SIPService) SyncCatalogChannels(ctx context.Context, nvrDeviceCode string, channels []ChannelInfo) error {
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, nvrDeviceCode)
	if err != nil {
		return fmt.Errorf("find nvr device: %w", err)
	}

	createdCount := 0
	updatedCount := 0
	for _, ch := range channels {
		if ch.DeviceID == "" {
			continue
		}
		status := MapChannelStatus(ch.Status)
		existing, err := s.deviceRepo.FindByGB28181DeviceID(ctx, ch.DeviceID)
		if err == nil && existing != nil {
			existing.DeviceName = ch.Name
			existing.Status = status
			existing.Manufacturer = ch.Manufacturer
			existing.Model = ch.Model
			existing.ParentNvrID = &gbDevice.ID
			if err := s.deviceRepo.Update(ctx, existing); err != nil {
				zap.L().Warn("update channel device failed", zap.String("channel_id", ch.DeviceID), zap.Error(err))
				continue
			}
			updatedCount++
			continue
		}
		newDev := &model.Device{
			DeviceName: ch.Name, AccessType: model.DeviceAccessTypeGB28181,
			GB28181DeviceID: ch.DeviceID, GB28181ChannelID: ch.DeviceID,
			Manufacturer: ch.Manufacturer, Model: ch.Model,
			Status: status, Enabled: true, ParentNvrID: &gbDevice.ID,
		}
		if err := s.deviceRepo.Create(ctx, newDev); err != nil {
			zap.L().Warn("create channel device failed", zap.String("channel_id", ch.DeviceID), zap.Error(err))
			continue
		}
		createdCount++
	}

	zap.L().Info("catalog channels synced",
		zap.String("nvr", nvrDeviceCode),
		zap.Int("total", len(channels)),
		zap.Int("created", createdCount),
		zap.Int("updated", updatedCount))

	if err := s.gbDeviceRepo.Update(ctx, gbDevice.ID, map[string]interface{}{"channel_count": len(channels)}); err != nil {
		zap.L().Warn("update gb28181 device channel count failed", zap.Error(err))
	}

	if s.cache != nil && s.hub != nil {
		deviceTaskKey := "catalog_task:device:" + nvrDeviceCode
		if taskIDBytes, err := s.cache.Get(ctx, deviceTaskKey); err == nil && len(taskIDBytes) > 0 {
			taskID := string(taskIDBytes)
			status := map[string]interface{}{"task_id": taskID, "status": "completed", "channel_count": len(channels)}
			statusData, _ := json.Marshal(status)
			if err := s.cache.Set(ctx, "catalog_task:"+taskID, statusData, 2*time.Minute); err != nil {
				zap.L().Warn("set catalog task status cache failed", zap.Error(err))
			}
			s.hub.Broadcast(&ws.Message{Type: "gb28181_catalog_completed", Payload: map[string]interface{}{"deviceCode": nvrDeviceCode, "success": true, "channelCount": len(channels)}})
		}
	}
	return nil
}
// alarmDispatchPayload is the payload for alarm dispatch tasks.
type alarmDispatchPayload struct {
	SmartRecordID string `json:"smart_record_id"`
	DeviceID      string `json:"device_id"`
	DeviceName    string `json:"device_name"`
	AlgorithmName string `json:"algorithm_name"`
	AlarmType     string `json:"alarm_type"`
	AlarmLevel    string `json:"alarm_level"`
	CaptureTime   string `json:"capture_time"`
	SnapshotURL   string `json:"snapshot_url"`
	RawResult     string `json:"raw_result,omitempty"`
}

// HandleAlarm processes a GB28181 alarm event from a device.
func (s *SIPService) HandleAlarm(ctx context.Context, alarm AlarmInfo) error {
	if alarm.DeviceID == "" {
		return errors.New(errors.ErrAlarmDeviceRequired, "")
	}
	now := time.Now()
	record := &model.SmartRecord{
		RecordID: uuid.NewString(), RecordType: model.RecordTypeAlarm,
		CaptureTime: now, AlarmType: alarm.AlarmType, AlarmLevel: alarm.AlarmLevel,
	}
	if s.gbDeviceRepo != nil {
		if gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, alarm.DeviceID); err == nil {
			if gbDev.DeviceID != nil {
				record.DeviceID = gbDev.DeviceID
			}
			record.DeviceName = gbDev.DeviceCode
		}
	}
	rawJSON, _ := json.Marshal(alarm)
	record.RawResult = datatypes.JSON(rawJSON)

	// Persist alarm record
	if s.smartRecordRepo != nil {
		if err := s.smartRecordRepo.Create(ctx, record); err != nil {
			zap.L().Warn("save GB28181 alarm to smart_records failed",
				zap.String("device_id", alarm.DeviceID), zap.Error(err))
			// Continue to enqueue dispatch even if persistence fails (old behavior)
		}
	}

	// Enqueue alarm dispatch task (always try, regardless of persistence outcome)
	if s.taskClient != nil {
		payload := alarmDispatchPayload{
			SmartRecordID: record.RecordID,
			DeviceID:      alarm.DeviceID,
			DeviceName:    record.DeviceName,
			AlarmType:     record.AlarmType,
			AlarmLevel:    record.AlarmLevel,
			CaptureTime:   record.CaptureTime.Format(time.RFC3339),
			SnapshotURL:   record.SnapshotImageURL,
			RawResult:     string(record.RawResult),
		}
		if err := s.taskClient.Enqueue(ctx, "alarm:dispatch", payload); err != nil {
			zap.L().Error("failed to enqueue alarm dispatch task",
				zap.String("record_id", record.RecordID), zap.Error(err))
		}
	}
	return nil
}

// buildSSRC constructs an SSRC string from the device code, using a prefix character.
// Ensures safe slicing even if deviceCode is shorter than expected.
func buildSSRC(deviceCode, prefix string) string {
	if len(deviceCode) < 10 {
		return prefix + deviceCode
	}
	return prefix + deviceCode[10:19]
}
