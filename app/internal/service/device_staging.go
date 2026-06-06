package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/niko-admin/niko-admin/internal/model"
)

type discoveredDeviceRepo interface {
	Upsert(ctx context.Context, item *model.DiscoveredDevice) error
	List(ctx context.Context, source, status, keyword string, page, pageSize int) ([]model.DiscoveredDevice, int64, error)
	BatchUpdateStatus(ctx context.Context, ids []string, status string) error
	FindByID(ctx context.Context, id string) (*model.DiscoveredDevice, error)
	MarkImported(ctx context.Context, id, deviceID string) error
}

type DeviceStagingService struct {
	repo       discoveredDeviceRepo
	deviceRepo deviceRepo
}

func NewDeviceStagingService(repo discoveredDeviceRepo, devRepo deviceRepo) *DeviceStagingService {
	return &DeviceStagingService{
		repo:       repo,
		deviceRepo: devRepo,
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

func (s *DeviceStagingService) BatchImport(ctx context.Context, ids []string) error {
	var errs []string
	for _, id := range ids {
		if err := s.ImportSingle(ctx, id); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", id, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("partial import failures: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (s *DeviceStagingService) ImportSingle(ctx context.Context, id string) error {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if item.Status == model.StatusImported {
		return nil
	}

	// 转化为 Device 并保存
	device := &model.Device{
		DeviceName:      item.DeviceName,
		AccessType:      item.AccessType,
		RtspURL:         item.AccessURL,
		GB28181DeviceID: item.GB28181Code,
		Manufacturer:    item.Manufacturer,
		Model:           item.Model,
		FirmwareVersion: item.FirmwareVersion,
		Status:          model.DeviceStatusUnknown,
		Enabled:         true,
	}

	// 补全默认名称
	if device.DeviceName == "" {
		if item.DeviceIP != "" {
			device.DeviceName = "Device_" + item.DeviceIP
		} else {
			device.DeviceName = "Discovered_" + id[:8]
		}
	}

	// 设置 ExternalKey 用于唯一约束去重
	switch device.AccessType {
	case model.DeviceAccessTypeRTSP:
		if device.RtspURL != "" {
			key := "rtsp:" + device.RtspURL
			device.ExternalKey = &key
		}
	case model.DeviceAccessTypeGB28181:
		if device.GB28181DeviceID != "" {
			key := "gb28181:" + device.GB28181DeviceID + ":" + device.GB28181ChannelID
			device.ExternalKey = &key
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

	return s.repo.MarkImported(ctx, id, device.ID)
}

