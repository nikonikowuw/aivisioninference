package service

import (
	"context"
	"fmt"
	"time"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

type GB28181PlatformConfigService struct {
	configRepo *repository.GB28181PlatformConfigRepository
	notifier   ConfigChangeNotifier
}

type ConfigChangeNotifier interface {
	NotifyConfigChange(ctx context.Context, needRestart bool)
}

func NewGB28181PlatformConfigService(
	configRepo *repository.GB28181PlatformConfigRepository,
) *GB28181PlatformConfigService {
	return &GB28181PlatformConfigService{
		configRepo: configRepo,
	}
}

func (s *GB28181PlatformConfigService) SetNotifier(notifier ConfigChangeNotifier) {
	s.notifier = notifier
}

func (s *GB28181PlatformConfigService) GetConfig(ctx context.Context) (*model.GB28181PlatformConfig, error) {
	return s.configRepo.Get(ctx)
}

func (s *GB28181PlatformConfigService) UpdateConfig(ctx context.Context, req dto.GB28181PlatformConfigRequest) (*model.GB28181PlatformConfig, error) {
	cfg, err := s.configRepo.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Determine if restart is required
	needRestart := false
	if cfg.ListenIP != req.ListenIP ||
		cfg.ListenPort != req.ListenPort ||
		cfg.Transport != req.Transport ||
		cfg.Enabled != req.Enabled {
		needRestart = true
	}

	cfg.Enabled = req.Enabled
	cfg.SipID = req.SipID
	cfg.SipDomain = req.SipDomain
	cfg.SipRealm = req.SipRealm
	if req.SipPassword != "" {
		cfg.SipPassword = req.SipPassword
	}
	cfg.ListenIP = req.ListenIP
	cfg.ListenPort = req.ListenPort
	cfg.Transport = req.Transport
	cfg.AdvertisedIP = req.AdvertisedIP
	cfg.RtpIP = req.RtpIP
	cfg.HeartbeatTimeout = req.HeartbeatTimeout
	cfg.CatalogInterval = req.CatalogInterval
	cfg.UpdatedAt = time.Now()

	if err := s.configRepo.Update(ctx, cfg); err != nil {
		return nil, fmt.Errorf("update platform config: %w", err)
	}

	if s.notifier != nil {
		s.notifier.NotifyConfigChange(ctx, needRestart)
	}

	return cfg, nil
}
