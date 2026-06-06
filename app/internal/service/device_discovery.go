package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/nvr"
	"github.com/niko-admin/niko-admin/internal/pkg/onvif"
)

type DeviceDiscoveryService struct {
	stagingSvc   *DeviceStagingService
	onvifScanner *onvif.Scanner
	nvrExplorer  nvr.Explorer
}

func NewDeviceDiscoveryService(
	stagingSvc *DeviceStagingService,
	onvifScanner *onvif.Scanner,
	nvrExplorer nvr.Explorer,
) *DeviceDiscoveryService {
	if nvrExplorer == nil {
		nvrExplorer = &nvr.DefaultExplorer{}
	}
	return &DeviceDiscoveryService{
		stagingSvc:   stagingSvc,
		onvifScanner: onvifScanner,
		nvrExplorer:  nvrExplorer,
	}
}

func (s *DeviceDiscoveryService) DiscoverNVRChannels(ctx context.Context, device *model.Device) error {
	channels, err := s.nvrExplorer.ListChannels(ctx, device)
	if err != nil {
		return err
	}

	for _, ch := range channels {
		if err := s.stagingSvc.AddDiscovered(ctx, &model.DiscoveredDevice{
			Source:      model.SourceNVR,
			DeviceName:  ch.ChannelName,
			AccessType:  "nvr_channel",
			AccessURL:   ch.AccessURL,
			NvrDeviceID: &device.ID,
			Status:      model.StatusPending,
		}); err != nil {
			zap.L().Warn("NVR channel AddDiscovered failed",
				zap.String("channel", ch.ChannelName),
				zap.Error(err))
		}
	}

	return nil
}


func (s *DeviceDiscoveryService) DiscoverONVIF(ctx context.Context, netInterface string) error {
	devices, err := s.onvifScanner.Scan(ctx, netInterface, 5*time.Second)
	if err != nil {
		return err
	}

	for _, d := range devices {
		if err := s.stagingSvc.AddDiscovered(ctx, &model.DiscoveredDevice{
			Source:       model.SourceONVIF,
			DeviceIP:     d.IP,
			DeviceMAC:    d.MAC,
			Manufacturer: d.Manufacturer,
			Model:        d.Model,
			AccessType:   "rtsp",
			AccessURL:    d.StreamURL,
			Status:       model.StatusPending,
		}); err != nil {
			zap.L().Warn("ONVIF device AddDiscovered failed",
				zap.String("ip", d.IP),
				zap.Error(err))
		}
	}

	return nil
}
