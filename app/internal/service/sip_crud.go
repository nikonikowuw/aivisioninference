package service

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
)

// CreateGB28181Device creates a new GB28181 device
func (s *SIPService) CreateGB28181Device(ctx context.Context, req dto.GB28181DeviceCreateRequest) (*model.GB28181Device, error) {
	sipID := req.SipID
	if sipID == "" {
		sipID = req.DeviceCode
	}
	heartbeat := req.HeartbeatInterval
	if heartbeat == 0 {
		heartbeat = 60
	}
	device := &model.GB28181Device{
		DeviceCode:        req.DeviceCode,
		SipID:             sipID,
		SipDomain:         req.SipDomain,
		SipPassword:       req.SipPassword,
		HeartbeatInterval: heartbeat,
		Status:            model.GB28181StatusOffline,
	}
	if err := s.gbDeviceRepo.Create(ctx, device); err != nil {
		return nil, err
	}
	return device, nil
}

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

// BatchDeleteGB28181Devices 批量删除 GB28181 设备，逐条删除并汇总结果。
// 注意：此方法支持部分成功语义，返回结果中会明确标记每条记录的成功/失败状态。
func (s *SIPService) BatchDeleteGB28181Devices(ctx context.Context, ids []string) dto.BatchResult {
	var result dto.BatchResult
	result.Total = len(ids)
	for _, id := range ids {
		if err := s.gbDeviceRepo.Delete(ctx, id); err != nil {
			result.Items = append(result.Items, dto.BatchItemResult{ID: id, Success: false, Message: err.Error()})
			result.Failed++
		} else {
			result.Items = append(result.Items, dto.BatchItemResult{ID: id, Success: true})
			result.Success++
		}
	}
	return result
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

// ListNVRs 列出 GB28181 NVR 设备（从统一 Device 表查询）
func (s *SIPService) ListNVRs(ctx context.Context, keyword, status string, page, pageSize int) ([]dto.GB28181DeviceResponse, int64, error) {
	if s.deviceRepoV2 == nil {
		return []dto.GB28181DeviceResponse{}, 0, nil
	}
	items, total, err := s.deviceRepoV2.FindGB28181NVRs(ctx, keyword, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 解决 N+1 问题
	var deviceCodes []string
	for _, item := range items {
		deviceCodes = append(deviceCodes, item.GB28181DeviceID)
	}

	cfgMap := make(map[string]*model.DeviceSipConfig)
	if len(deviceCodes) > 0 && s.deviceSipConfigRepo != nil {
		cfgs, _ := s.deviceSipConfigRepo.FindByDeviceCodes(ctx, deviceCodes)
		for i := range cfgs {
			cfgMap[cfgs[i].DeviceCode] = &cfgs[i]
		}
	}

	var list []dto.GB28181DeviceResponse
	for _, item := range items {
		resp := dto.GB28181DeviceResponse{
			ID:           item.ID,
			DeviceCode:   item.GB28181DeviceID,
			Manufacturer: item.Manufacturer,
			Model:        item.Model,
			Firmware:     item.FirmwareVersion,
			Status:       item.Status,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		}
		if cfg, ok := cfgMap[item.GB28181DeviceID]; ok {
			resp.RegisterAddress = cfg.RegisterAddress
			resp.SipID = cfg.SipID
			resp.SipDomain = cfg.SipDomain
			resp.HeartbeatInterval = cfg.HeartbeatInterval
			resp.ChannelCount = cfg.ChannelCount
		}
		list = append(list, resp)
	}
	return list, total, nil
}

// GetNVRChannels 查询指定 NVR 下的所有通道
func (s *SIPService) GetNVRChannels(ctx context.Context, nvrID string, page, pageSize int) ([]dto.GB28181DeviceResponse, int64, error) {
	if s.deviceRepoV2 == nil {
		return []dto.GB28181DeviceResponse{}, 0, nil
	}
	channels, total, err := s.deviceRepoV2.FindGB28181Channels(ctx, nvrID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	var list []dto.GB28181DeviceResponse
	for _, ch := range channels {
		list = append(list, dto.GB28181DeviceResponse{
			ID:           ch.ID,
			DeviceCode:   ch.GB28181DeviceID,
			Manufacturer: ch.Manufacturer,
			Model:        ch.Model,
			Firmware:     ch.FirmwareVersion,
			Status:       ch.Status,
			CreatedAt:    ch.CreatedAt,
			UpdatedAt:    ch.UpdatedAt,
		})
	}
	return list, total, nil
}
