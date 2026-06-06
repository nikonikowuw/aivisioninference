package service

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
)

// ListGB28181Devices returns a paginated list of GB28181 devices
func (s *SIPService) ListGB28181Devices(ctx context.Context, req dto.GB28181DeviceListRequest) ([]model.GB28181Device, int64, error) {
	return s.gbDeviceRepo.List(ctx, req.Keyword, req.Status, req.GetPage(), req.GetPageSize())
}

// GetGB28181DeviceByID returns a GB28181 device by ID
func (s *SIPService) GetGB28181DeviceByID(ctx context.Context, id string) (*model.GB28181Device, error) {
	return s.gbDeviceRepo.FindByID(ctx, id)
}

// UpdateGB28181Device updates a GB28181 device
func (s *SIPService) UpdateGB28181Device(ctx context.Context, id string, req dto.GB28181DeviceUpdateRequest) (*model.GB28181Device, error) {
	_, err := s.gbDeviceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.SipID != nil {
		updates["sip_id"] = *req.SipID
	}
	if req.SipDomain != nil {
		updates["sip_domain"] = *req.SipDomain
	}
	if req.SipPassword != nil && *req.SipPassword != "" {
		updates["sip_password"] = *req.SipPassword
	}
	if req.HeartbeatInterval != nil {
		updates["heartbeat_interval"] = *req.HeartbeatInterval
	}

	if len(updates) > 0 {
		if err := s.gbDeviceRepo.Update(ctx, id, updates); err != nil {
			return nil, err
		}
	}

	return s.gbDeviceRepo.FindByID(ctx, id)
}

// DeleteGB28181Device deletes a GB28181 device
func (s *SIPService) DeleteGB28181Device(ctx context.Context, id string) error {
	return s.gbDeviceRepo.Delete(ctx, id)
}

// GetGB28181DeviceChannels returns channels of a GB28181 device
func (s *SIPService) GetGB28181DeviceChannels(ctx context.Context, id string) ([]model.Device, error) {
	// parentNvrID is the ID of the Device record that corresponds to this GB28181Device.
	// Note: NVR channels have ParentNvrID = Device.ID, where Device.GB28181DeviceID = GB28181Device.DeviceCode
	gbDevice, err := s.gbDeviceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	device, err := s.deviceRepo.FindByGB28181DeviceID(ctx, gbDevice.DeviceCode)
	if err != nil {
		return nil, err
	}

	return s.deviceRepo.FindByParentNvrID(ctx, device.ID)
}
