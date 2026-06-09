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

	// GB28181 ZLM 集成字段
	zlmClient     *zlm.Client
	streamManager *StreamManager
	zlmBaseIP     string // ZLM 对设备可见的 IP（设备推流目标）
	rtmpPort      int    // ZLM RTMP 端口
	rtspPort      int    // ZLM RTSP 端口
	httpPort      int    // ZLM HTTP 端口

	// 缓存和 WebSocket
	cache pkgcache.Cache
	hub   *ws.Hub
}

// NewSIPService creates a new SIPService.
func NewSIPService(
	deviceRepo *repository.DeviceRepository,
	gbDeviceRepo *repository.GB28181DeviceRepository,
	mediaStreamRepo *repository.MediaStreamRepository,
) *SIPService {
	return &SIPService{
		deviceRepo:      deviceRepo,
		gbDeviceRepo:    gbDeviceRepo,
		mediaStreamRepo: mediaStreamRepo,
	}
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
	}
}

func (s *SIPService) SetDiscoveryService(svc *DeviceDiscoveryService) {
	s.discoverySvc = svc
}

var gb28181DeviceIDRegex = regexp.MustCompile(`^\d{20}$`)

func (s *SIPService) ValidateDeviceCode(deviceID string) error {
	if !gb28181DeviceIDRegex.MatchString(deviceID) {
		return errors.New(errors.ErrInvalidGB28181DeviceCode, "")
	}
	return nil
}

// HandleRegister validates a device SIP registration request.
// Returns error if registration should be rejected.
func (s *SIPService) HandleRegister(ctx context.Context, deviceID, remoteIP string, port int) error {
	// 1. Validate 20-digit GB28181 device code format
	if err := s.ValidateDeviceCode(deviceID); err != nil {
		return err
	}

	// 2. Find the device in database
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return errors.New(errors.ErrDeviceNotFound, "")
	}

	// 3. Update registration info
	now := time.Now()
	gbDevice.RegisterAddress = remoteIP
	gbDevice.RegisterPort = port
	gbDevice.LastRegisterAt = &now
	gbDevice.Status = model.GB28181StatusOnline

	if err := s.gbDeviceRepo.UpdateStatus(ctx, gbDevice.ID, model.GB28181StatusOnline); err != nil {
		return fmt.Errorf("update device status: %w", err)
	}

	// 4. 同步到新数据模型 DeviceSipConfig
	if s.deviceSipConfigRepo != nil {
		if err := s.deviceSipConfigRepo.UpdateRegisterAddress(ctx, deviceID, remoteIP, port); err != nil {
			zap.L().Warn("update DeviceSipConfig register address failed", zap.String("device_code", deviceID), zap.Error(err))
		}
	}

	// 5. 同步到 Device 模型（如果有关联记录）
	if gbDevice.DeviceID != nil {
		_ = s.deviceRepo.UpdateStatus(ctx, *gbDevice.DeviceID, model.DeviceStatusOnline, "", "")
	} else {
		// 无关联 Device 记录时，通过 ExternalKey 查找
		key := "gb28181_nvr:" + deviceID
		if dev, err := s.deviceRepo.FindByExternalKey(ctx, key); err == nil && dev != nil {
			_ = s.deviceRepo.UpdateStatus(ctx, dev.ID, model.DeviceStatusOnline, "", "")
		}
	}

	// 6. 触发目录查询（如果是 NVR/平台）
	// 使用独立 context，HTTP 请求结束后 goroutine 仍可正常执行
	go func() {
		if err := s.QueryCatalog(context.Background(), gbDevice.DeviceCode); err != nil {
			zap.L().Warn("catalog query failed", zap.String("device_code", gbDevice.DeviceCode), zap.Error(err))
		}
	}()

	return nil
}

// HandleHeartbeat updates the heartbeat timestamp for a registered device.
func (s *SIPService) HandleHeartbeat(ctx context.Context, deviceID string) error {
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return errors.New(errors.ErrDeviceNotFound, "")
	}

	if err := s.gbDeviceRepo.UpdateHeartbeat(ctx, gbDevice.ID); err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
	}

	// 同步到 DeviceSipConfig
	if s.deviceSipConfigRepo != nil {
		if err := s.deviceSipConfigRepo.UpdateHeartbeat(ctx, deviceID); err != nil {
			zap.L().Debug("update DeviceSipConfig heartbeat failed", zap.String("device_code", deviceID), zap.Error(err))
		}
	}

	if gbDevice.Status != model.GB28181StatusOnline {
		if err := s.gbDeviceRepo.UpdateStatus(ctx, gbDevice.ID, model.GB28181StatusOnline); err != nil {
			return fmt.Errorf("update status to online: %w", err)
		}
	}

	return nil
}

// CheckHeartbeatTimeout scans for devices that have timed out and marks them offline.
func (s *SIPService) CheckHeartbeatTimeout(ctx context.Context, timeout time.Duration) ([]model.GB28181Device, error) {
	offlineDevices, err := s.gbDeviceRepo.FindOfflineDevices(ctx, timeout)
	if err != nil {
		return nil, fmt.Errorf("find offline devices: %w", err)
	}

	for _, dev := range offlineDevices {
		if err := s.gbDeviceRepo.UpdateStatus(ctx, dev.ID, model.GB28181StatusOffline); err != nil {
			return nil, fmt.Errorf("update device %s to offline: %w", dev.DeviceCode, err)
		}

		// Sync to Device model
		if dev.DeviceID != nil {
			_ = s.deviceRepo.UpdateStatus(ctx, *dev.DeviceID, model.DeviceStatusOffline, "", "")
		}
	}

	return offlineDevices, nil
}

// BuildCatalogueResponse generates a GB28181 MANSCDP XML catalogue response.
// deviceID: the SIP device ID that was queried.
// sn: the SIP message sequence number from the query.
func (s *SIPService) BuildCatalogueResponse(ctx context.Context, deviceID, sn string) (string, error) {
	// Validate device code
	if err := s.ValidateDeviceCode(deviceID); err != nil {
		return "", err
	}

	// Find the device
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return "", errors.New(errors.ErrDeviceNotFound, "")
	}

	// Find the associated system device
	status := "ON"
	if gbDevice.Status == model.GB28181StatusOffline {
		status = "OFF"
	}

	// 构建通道列表，转义 XML 特殊字符防止注入
	escapedSN := html.EscapeString(sn)
	escapedDeviceID := html.EscapeString(deviceID)
	escapedName := html.EscapeString(gbDevice.DeviceCode)
	escapedManufacturer := html.EscapeString(gbDevice.Manufacturer)
	escapedModel := html.EscapeString(gbDevice.Model)

	catalogXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
</Response>`, escapedSN, escapedDeviceID, escapedDeviceID, escapedName, escapedManufacturer, escapedModel, status, 0.0, 0.0)

	return catalogXML, nil
}

// ====== GB28181 业务扩展（依赖 ZLM） ======

// HandleUnregister 处理设备主动注销（Expires=0）
func (s *SIPService) HandleUnregister(ctx context.Context, deviceID string) error {
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return errors.New(errors.ErrDeviceNotFound, "")
	}
	if err := s.gbDeviceRepo.UpdateStatus(ctx, gbDevice.ID, model.GB28181StatusOffline); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if gbDevice.DeviceID != nil {
		_ = s.deviceRepo.UpdateStatus(ctx, *gbDevice.DeviceID, model.DeviceStatusOffline, "", "")
	}
	zap.L().Info("GB28181 device unregistered", zap.String("device_id", deviceID))
	return nil
}

// QueryCatalog 主动查询设备目录
func (s *SIPService) QueryCatalog(ctx context.Context, deviceCode string) error {
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
	if err != nil {
		return errors.New(errors.ErrDeviceNotFound, "")
	}
	if gbDevice.Status != model.GB28181StatusOnline {
		return errors.New(errors.ErrGB28181DeviceOffline, "")
	}
	// ZLM 内部 SIP 栈处理目录查询
	// Go 端通过 webhook 接收响应后调用 SyncCatalogChannels
	zap.L().Info("catalog query requested",
		zap.String("device_code", deviceCode),
		zap.String("device_ip", gbDevice.RegisterAddress))
	return nil
}

// StartLiveStream 通过 ZLM 控制 GB28181 设备向平台推流
func (s *SIPService) StartLiveStream(ctx context.Context, deviceCode, streamID string) (string, error) {
	if s.zlmClient == nil {
		return "", errors.New(errors.ErrZLMNotConfigured, "")
	}
	// 1. ZLM 创建 RTP 接收端口
	port, err := s.zlmClient.OpenRtpServer(ctx, zlm.OpenRtpServerRequest{
		Port:     0,
		TCPMode:  0,
		StreamID: streamID,
	})
	if err != nil {
		return "", fmt.Errorf("open rtp server: %w", err)
	}
	// 2. ZLM 触发设备推流
	_, err = s.zlmClient.StartSendRtp(ctx, zlm.StartSendRtpRequest{
		Vhost:   "__defaultVhost__",
		App:     "live",
		Stream:  streamID,
		SSRC:    "1",
		DstURL:  s.zlmBaseIP,
		DstPort: port,
		IsUDP:   1,
	})
	if err != nil {
		_ = s.zlmClient.CloseRtpServer(ctx, streamID)
		return "", fmt.Errorf("start send rtp: %w", err)
	}
	// 3. StreamManager 注册（以 deviceCode 为 key，ZLM 流 ID 作 metadata）
	if s.streamManager != nil {
		_ = s.streamManager.Acquire(ctx, deviceCode, "gb28181_live", map[string]string{
			"zlm_stream_id": streamID,
			"device_code":   deviceCode,
		})
	}
	return fmt.Sprintf("http://%s:%d/live/%s/hls.m3u8", s.zlmBaseIP, s.httpPort, streamID), nil
}

// StopLiveStream 停止 GB28181 实时预览
// deviceCode: NVR 设备国标编码（作为 StreamManager key），streamID: ZLM 内部流 ID
func (s *SIPService) StopLiveStream(ctx context.Context, deviceCode, streamID string) error {
	if s.zlmClient == nil {
		return errors.New(errors.ErrZLMNotConfigured, "")
	}
	_ = s.zlmClient.StopSendRtp(ctx, zlm.StopSendRtpRequest{
		Vhost:  "__defaultVhost__",
		App:    "live",
		Stream: streamID,
	})
	_ = s.zlmClient.CloseRtpServer(ctx, streamID)
	if s.streamManager != nil {
		_ = s.streamManager.Release(ctx, deviceCode, "gb28181_live")
	}
	return nil
}

// StartPlayback 发起 GB28181 录像回放
func (s *SIPService) StartPlayback(ctx context.Context, deviceCode, streamID string, start, end time.Time) (string, error) {
	if err := s.ValidateDeviceCode(deviceCode); err != nil {
		return "", err
	}
	if end.Before(start) {
		return "", errors.New(errors.ErrTimeRangeOrder, "")
	}
	if s.zlmClient == nil {
		return "", errors.New(errors.ErrZLMNotConfigured, "")
	}

	// 1. ZLM 创建 RTP 接收端口
	port, err := s.zlmClient.OpenRtpServer(ctx, zlm.OpenRtpServerRequest{
		Port:     0,
		TCPMode:  0,
		StreamID: streamID,
	})
	if err != nil {
		return "", fmt.Errorf("open rtp server: %w", err)
	}

	// 2. ZLM 触发设备回放推流
	_, err = s.zlmClient.StartSendRtp(ctx, zlm.StartSendRtpRequest{
		Vhost:   "__defaultVhost__",
		App:     "rtp",
		Stream:  streamID,
		SSRC:    "1",
		DstURL:  s.zlmBaseIP,
		DstPort: port,
		IsUDP:   1,
	})
	if err != nil {
		_ = s.zlmClient.CloseRtpServer(ctx, streamID)
		return "", fmt.Errorf("start playback rtp: %w", err)
	}

	// 3. StreamManager 注册
	if s.streamManager != nil {
		if err := s.streamManager.Acquire(ctx, deviceCode, "gb28181_playback", map[string]string{
			"zlm_stream_id": streamID,
			"device_code":   deviceCode,
		}); err != nil {
			// 注册失败，回滚 ZLM 资源
			_ = s.zlmClient.StopSendRtp(ctx, zlm.StopSendRtpRequest{
				Vhost:  "__defaultVhost__",
				App:    "rtp",
				Stream: streamID,
			})
			_ = s.zlmClient.CloseRtpServer(ctx, streamID)
			zap.L().Warn("StreamManager Acquire failed, rolled back ZLM resources",
				zap.String("device", deviceCode), zap.String("stream", streamID), zap.Error(err))
			return "", fmt.Errorf("stream manager acquire: %w", err)
		}
	}

	zap.L().Info("GB28181 playback started",
		zap.String("device", deviceCode),
		zap.String("stream", streamID),
		zap.Time("start", start),
		zap.Time("end", end))

	return fmt.Sprintf("http://%s:%d/rtp/%s.flv", s.zlmBaseIP, s.httpPort, streamID), nil
}

// PlaybackControl 回放控制
func (s *SIPService) PlaybackControl(ctx context.Context, streamID, action string, speed float64, stamp int64) error {
	if s.zlmClient == nil {
		return errors.New(errors.ErrZLMNotConfigured, "")
	}
	switch action {
	case "scale":
		return s.zlmClient.SetRecordSpeed(ctx, zlm.SetRecordSpeedRequest{
			Vhost:  "__defaultVhost__",
			App:    "live",
			Stream: streamID,
			Speed:  speed,
		})
	case "seek":
		return s.zlmClient.SeekRecordStamp(ctx, zlm.SeekRecordStampRequest{
			Vhost:  "__defaultVhost__",
			App:    "live",
			Stream: streamID,
			Stamp:  stamp,
		})
	default:
		return errors.New(errors.ErrInvalidPlaybackAction, "")
	}
}

// SyncCatalogChannels 同步目录响应到 Device 表（upsert）
func (s *SIPService) SyncCatalogChannels(ctx context.Context, nvrDeviceCode string, channels []ChannelInfo) error {
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, nvrDeviceCode)
	if err != nil {
		return errors.New(errors.ErrDeviceNotFound, "")
	}
	successCount := 0
	updatedCount := 0
	for _, ch := range channels {
		if ch.DeviceID == "" {
			continue
		}
		deviceStatus := MapChannelStatus(ch.Status)
		// 先查现有设备，避免重复创建
		existing, err := s.deviceRepo.FindByGB28181DeviceID(ctx, ch.DeviceID)
		if err == nil && existing != nil {
			existing.DeviceName = ch.Name
			existing.Status = deviceStatus
			existing.Manufacturer = ch.Manufacturer
			existing.Model = ch.Model
			existing.ParentNvrID = &gbDevice.ID
			if uerr := s.deviceRepo.Update(ctx, existing); uerr != nil {
				zap.L().Warn("update channel device failed",
					zap.String("channel_id", ch.DeviceID), zap.Error(uerr))
				continue
			}
			updatedCount++
			continue
		}
		// 创建新设备
		newDevice := &model.Device{
			DeviceName:       ch.Name,
			AccessType:       model.DeviceAccessTypeGB28181,
			GB28181DeviceID:  ch.DeviceID,
			GB28181ChannelID: ch.DeviceID,
			Manufacturer:     ch.Manufacturer,
			Model:            ch.Model,
			Status:           deviceStatus,
			Enabled:          true,
			ParentNvrID:      &gbDevice.ID,
		}
		if err := s.deviceRepo.Create(ctx, newDevice); err != nil {
			zap.L().Warn("create channel device failed",
				zap.String("channel_id", ch.DeviceID), zap.Error(err))
			continue
		}
		successCount++
	}
	zap.L().Info("catalog channels synced",
		zap.String("nvr", nvrDeviceCode),
		zap.Int("total", len(channels)),
		zap.Int("created", successCount),
		zap.Int("updated", updatedCount))

	// 更新 GB28181Device 的通道数
	if uerr := s.gbDeviceRepo.Update(ctx, gbDevice.ID, map[string]interface{}{
		"channel_count": len(channels),
	}); uerr != nil {
		zap.L().Warn("update gb28181 device channel count failed", zap.Error(uerr))
	}

	// 更新任务状态和触发 WebSocket 广播
	if s.cache != nil && s.hub != nil {
		deviceTaskKey := "catalog_task:device:" + nvrDeviceCode
		if taskIDBytes, err := s.cache.Get(ctx, deviceTaskKey); err == nil && len(taskIDBytes) > 0 {
			taskID := string(taskIDBytes)
			status := map[string]interface{}{
				"task_id":       taskID,
				"status":        "completed",
				"channel_count": len(channels),
			}
			statusData, _ := json.Marshal(status)
			_ = s.cache.Set(ctx, "catalog_task:"+taskID, statusData, 2*time.Minute)

			// 广播 ws 事件
			msg := &ws.Message{
				Type: "gb28181_catalog_completed",
				Payload: map[string]interface{}{
					"deviceCode":   nvrDeviceCode,
					"success":      true,
					"channelCount": len(channels),
				},
			}
			s.hub.Broadcast(msg)
		}
	}

	return nil
}

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

// enqueueAlarmDispatch 派发告警处理任务到队列
func (s *SIPService) enqueueAlarmDispatch(ctx context.Context, record *model.SmartRecord, alarm AlarmInfo) {
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

// HandleAlarm 处理设备告警上报
func (s *SIPService) HandleAlarm(ctx context.Context, alarm AlarmInfo) error {
	if alarm.DeviceID == "" {
		return errors.New(errors.ErrAlarmDeviceRequired, "")
	}
	zap.L().Info("GB28181 alarm received",
		zap.String("device_id", alarm.DeviceID),
		zap.String("alarm_type", alarm.AlarmType),
		zap.String("alarm_level", alarm.AlarmLevel))
	// 入库 smart_records（GB28181 告警），RecordType=alarm
	now := time.Now()
	record := &model.SmartRecord{
		RecordID:    uuid.NewString(), // 主键必填
		RecordType:  model.RecordTypeAlarm,
		CaptureTime: now,
		AlarmType:   alarm.AlarmType,
		AlarmLevel:  alarm.AlarmLevel,
	}
	// 关联设备（如已入库）
	if s.gbDeviceRepo != nil {
		if gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, alarm.DeviceID); err == nil {
			if gbDev.DeviceID != nil {
				record.DeviceID = gbDev.DeviceID
			}
			record.DeviceName = gbDev.DeviceCode
		}
	}
	// 序列化告警原文
	rawJSON, _ := json.Marshal(map[string]string{
		"device_id":   alarm.DeviceID,
		"alarm_type":  alarm.AlarmType,
		"alarm_level": alarm.AlarmLevel,
		"alarm_time":  alarm.AlarmTime,
	})
	record.RawResult = datatypes.JSON(rawJSON)
	if s.smartRecordRepo == nil {
		if s.taskClient != nil {
			s.enqueueAlarmDispatch(ctx, record, alarm)
		}
		return nil
	}

	if err := s.smartRecordRepo.Create(ctx, record); err != nil {
		zap.L().Warn("save GB28181 alarm to smart_records failed",
			zap.String("device_id", alarm.DeviceID), zap.Error(err))
		return nil
	}

	// 入库成功后派发告警任务
	if s.taskClient != nil {
		s.enqueueAlarmDispatch(ctx, record, alarm)
	}

	return nil
}
