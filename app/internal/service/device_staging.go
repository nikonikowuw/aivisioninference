package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	"github.com/niko-admin/niko-admin/internal/repository"
)

type discoveredDeviceRepo interface {
	Upsert(ctx context.Context, item *model.DiscoveredDevice) error
	List(ctx context.Context, source, status, keyword string, page, pageSize int) ([]model.DiscoveredDevice, int64, error)
	BatchUpdateStatus(ctx context.Context, ids []string, status string) error
	FindByID(ctx context.Context, id string) (*model.DiscoveredDevice, error)
	MarkImported(ctx context.Context, id, deviceID string) error
	ResetByDeviceID(ctx context.Context, deviceID string) error
	BatchResetByDeviceIDs(ctx context.Context, deviceIDs []string) error
}

type DeviceStagingService struct {
	repo                discoveredDeviceRepo
	deviceRepo          deviceRepo
	gbDeviceRepo        *repository.GB28181DeviceRepository
	deviceSipConfigRepo *repository.DeviceSipConfigRepository
}

func NewDeviceStagingService(
	repo discoveredDeviceRepo,
	devRepo deviceRepo,
	gbDeviceRepo *repository.GB28181DeviceRepository,
	deviceSipConfigRepo *repository.DeviceSipConfigRepository,
) *DeviceStagingService {
	return &DeviceStagingService{
		repo:                repo,
		deviceRepo:          devRepo,
		gbDeviceRepo:        gbDeviceRepo,
		deviceSipConfigRepo: deviceSipConfigRepo,
	}
}

func (s *DeviceStagingService) AddDiscovered(ctx context.Context, item *model.DiscoveredDevice) error {
	return s.repo.Upsert(ctx, item)
}

func (s *DeviceStagingService) List(ctx context.Context, source, status, keyword string, page, pageSize int) ([]model.DiscoveredDevice, int64, error) {
	return s.repo.List(ctx, source, status, keyword, page, pageSize)
}

func (s *DeviceStagingService) BatchIgnore(ctx context.Context, ids []string) error {
	return s.repo.BatchUpdateStatus(ctx, ids, model.StatusIgnored)
}

func (s *DeviceStagingService) BatchImport(ctx context.Context, ids []string, username, password string, enableInfer *bool) error {
	var errs []string
	for _, id := range ids {
		if err := s.ImportSingle(ctx, id, username, password, "", enableInfer); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", id, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("partial import failures: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (s *DeviceStagingService) ImportSingle(ctx context.Context, id string, username, password, deviceName string, enableInfer *bool) error {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if item.Status == model.StatusImported {
		return nil
	}

	// 验证凭证
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	// 转化为 Device 并保存
	device := &model.Device{
		DeviceName:      deviceName,
		AccessType:      item.AccessType,
		RtspURL:         item.AccessURL,
		Username:        username,
		Password:        password,
		GB28181DeviceID: item.GB28181Code,
		Manufacturer:    item.Manufacturer,
		Model:           item.Model,
		FirmwareVersion: item.FirmwareVersion,
		Status:          model.DeviceStatusUnknown,
		Enabled:         true,
	}

	// 设置自动推理
	if enableInfer != nil {
		autoInfer := *enableInfer
		device.AutoInfer = autoInfer
	} else {
		device.AutoInfer = true // 默认启用
	}

	// 补全默认名称
	if device.DeviceName == "" {
		if item.DeviceIP != "" {
			device.DeviceName = "Device_" + item.DeviceIP
		} else {
			device.DeviceName = "Discovered_" + id[:8]
		}
	}

	// 注入凭证到 RTSP URL
	if device.AccessType == model.DeviceAccessTypeRTSP && device.RtspURL != "" {
		device.RtspURL = injectCredentialsToRTSP(device.RtspURL, username, password)
	}

	// Hashing Unified Device password
	if password != "" {
		hashed, hashErr := hash.Hash(password)
		if hashErr == nil {
			device.Password = hashed
		}
	}

	// 设置 ExternalKey 用于唯一约束去重
	isGB28181 := device.AccessType == model.DeviceAccessTypeGB28181 || device.AccessType == model.DeviceAccessTypeGB28181NVR
	if isGB28181 {
		device.AccessType = model.DeviceAccessTypeGB28181NVR
		key := "gb28181_nvr:" + item.GB28181Code
		device.ExternalKey = &key
	} else {
		switch device.AccessType {
		case model.DeviceAccessTypeRTSP:
			if device.RtspURL != "" {
				key := "rtsp:" + device.RtspURL
				device.ExternalKey = &key
			}
		}
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		// 如果是因为 ExternalKey 冲突，尝试关联已有设备
		if device.ExternalKey != nil {
			if existing, findErr := s.deviceRepo.FindByExternalKey(ctx, *device.ExternalKey); findErr == nil {
				return s.repo.MarkImported(ctx, id, existing.ID)
			}
		}
		return err
	}

	// Create GB28181Device & DeviceSipConfig if it's GB28181
	if isGB28181 {
		if s.gbDeviceRepo != nil {
			// NOTE: SipPassword stores the raw password for SIP Digest auth (not bcrypt-hashed),
			// unlike device.Password which is bcrypt-hashed for login.
			// SIP digest auth needs the raw password to compute MD5(response).

			// Safe slice for SipDomain (GB28181Code is typically 20 chars)
			sipDomain := safeSipDomain(item.GB28181Code)

			gbDevice := &model.GB28181Device{
				DeviceID:          &device.ID,
				DeviceCode:        item.GB28181Code,
				SipID:             item.GB28181Code,
				SipDomain:         sipDomain,
				SipPassword:       password, // raw password for SIP digest auth
				HeartbeatInterval: 60,
				Status:            model.GB28181StatusOffline,
				Manufacturer:      item.Manufacturer,
				Model:             item.Model,
				Firmware:          item.FirmwareVersion,
				ExternalKey:       "gb28181:" + item.GB28181Code,
			}
			if existingGb, findErr := s.gbDeviceRepo.FindByDeviceCode(ctx, item.GB28181Code); findErr != nil || existingGb == nil {
				if err := s.gbDeviceRepo.Create(ctx, gbDevice); err != nil {
					zap.L().Warn("create gb28181 device failed", zap.String("code", item.GB28181Code), zap.Error(err))
				}
			} else {
				if err := s.gbDeviceRepo.Update(ctx, existingGb.ID, map[string]interface{}{
					"device_id":    &device.ID,
					"sip_password": password,
				}); err != nil {
					zap.L().Warn("update gb28181 device failed", zap.String("code", item.GB28181Code), zap.Error(err))
				}
			}
		}

		if s.deviceSipConfigRepo != nil {
			sipDomain := safeSipDomain(item.GB28181Code)

			sipConfig := &model.DeviceSipConfig{
				DeviceID:          device.ID,
				DeviceCode:        item.GB28181Code,
				SipID:             item.GB28181Code,
				SipDomain:         sipDomain,
				SipPassword:       password,
				HeartbeatInterval: 60,
			}
			if existingConfig, findErr := s.deviceSipConfigRepo.FindByDeviceCode(ctx, item.GB28181Code); findErr != nil || existingConfig == nil {
				if err := s.deviceSipConfigRepo.Create(ctx, sipConfig); err != nil {
					zap.L().Warn("create device sip config failed", zap.String("code", item.GB28181Code), zap.Error(err))
				}
			} else {
				if err := s.deviceSipConfigRepo.Update(ctx, existingConfig.ID, map[string]interface{}{
					"device_id":    device.ID,
					"sip_password": password,
				}); err != nil {
					zap.L().Warn("update device sip config failed", zap.String("code", item.GB28181Code), zap.Error(err))
				}
			}
		}
	}

	return s.repo.MarkImported(ctx, id, device.ID)
}

// injectCredentialsToRTSP 将凭证注入到 RTSP URL 中
// 输入: rtsp://192.168.1.100:554/Streaming/Channels/101
// 输出: rtsp://admin:password@192.168.1.100:554/Streaming/Channels/101
func injectCredentialsToRTSP(rtspURL, username, password string) string {
	const prefix = "rtsp://"
	if !strings.HasPrefix(rtspURL, prefix) {
		return rtspURL
	}

	// 移除已有凭证（如果有）并注入新的
	authority := rtspURL[len(prefix):]
	if idx := strings.Index(authority, "@"); idx >= 0 {
		authority = authority[idx+1:]
	}
	return prefix + username + ":" + password + "@" + authority
}

// safeSipDomain safely extracts the first 10 characters of a GB28181 code for use as SIP domain.
// Returns the full code if it's shorter than 10 characters to prevent panic.
func safeSipDomain(code string) string {
	if len(code) >= 10 {
		return code[:10]
	}
	return code
}

