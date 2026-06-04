package service

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"

	"github.com/niko-admin/niko-admin/internal/model"
)

// SIPService handles GB28181 SIP signaling logic.
type SIPService struct {
	deviceRepo      *repository.DeviceRepository
	gbDeviceRepo    *repository.GB28181DeviceRepository
	mediaStreamRepo *repository.MediaStreamRepository
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

var gb28181DeviceIDRegex = regexp.MustCompile(`^\d{20}$`)

// HandleRegister validates a device SIP registration request.
// Returns error if registration should be rejected.
func (s *SIPService) HandleRegister(ctx context.Context, deviceID, remoteIP string, port int) error {
	// 1. Validate 20-digit GB28181 device code format
	if !gb28181DeviceIDRegex.MatchString(deviceID) {
		return errors.New(errors.ErrBadRequest, fmt.Sprintf("invalid GB28181 device code format: %s", deviceID))
	}

	// 2. Find the device in database
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return errors.New(errors.ErrNotFound, fmt.Sprintf("GB28181 device not found: %s", deviceID))
	}

	// 子设备（通道）允许自动注册，但暂不处理
	if gbDevice == nil {
		return errors.New(errors.ErrNotFound, fmt.Sprintf("device not registered in system: %s", deviceID))
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

	// 4. Sync to Device model if associated
	if gbDevice.DeviceID != nil {
		_ = s.deviceRepo.UpdateStatus(ctx, *gbDevice.DeviceID, model.DeviceStatusOnline, "", "")
	}

	return nil
}

// HandleHeartbeat updates the heartbeat timestamp for a registered device.
func (s *SIPService) HandleHeartbeat(ctx context.Context, deviceID string) error {
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return errors.New(errors.ErrNotFound, fmt.Sprintf("device not found: %s", deviceID))
	}

	if err := s.gbDeviceRepo.UpdateHeartbeat(ctx, gbDevice.ID); err != nil {
		return fmt.Errorf("update heartbeat: %w", err)
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
	if !gb28181DeviceIDRegex.MatchString(deviceID) {
		return "", errors.New(errors.ErrBadRequest, fmt.Sprintf("invalid device code format: %s", deviceID))
	}

	// Find the device
	gbDevice, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return "", errors.New(errors.ErrNotFound, fmt.Sprintf("device not found: %s", deviceID))
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
