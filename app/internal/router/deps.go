package router

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/buildinfo"
	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/onvif"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

// RouteDeps 聚合路由注册阶段需要的 Handler、Service 与缓存依赖。
type RouteDeps struct {
	RBACCache            cache.Cache
	AuditService         *service.AuditService
	AuthHandler          *handler.AuthHandler
	WSHandler            *handler.WSHandler
	UserHandler          *handler.UserHandler
	RoleHandler          *handler.RoleHandler
	PermissionHandler    *handler.PermissionHandler
	FileHandler          *handler.FileHandler
	AuditHandler         *handler.AuditHandler
	TaskHandler          *handler.TaskHandler
	BrandHandler         *handler.BrandHandler
	MailHandler          *handler.MailHandler
	FeedbackHandler      *handler.FeedbackHandler
	DashboardHandler     *handler.DashboardHandler
	DeviceHandler        *handler.DeviceHandler
	DeviceGroupHandler   *handler.DeviceGroupHandler
	DeviceStagingHandler *handler.DeviceStagingHandler
	SystemHandler        *handler.SystemHandler
	StreamManager        *service.StreamManager
	LicenseHandler       *handler.LicenseHandler
	LicenseService       *service.LicenseService
	GB28181Handler       *handler.GB28181Handler
	MediaGB28181Handler  *handler.MediaGB28181Handler
	SmartRecordHandler   *handler.SmartRecordHandler
	GB28181ConfigHandler *handler.GB28181ConfigHandler
	SIPService           *service.SIPService
}

func provideAvatarStorage(cfg *Config) (*storage.LocalStorage, error) {
	return storage.NewLocalStorage(cfg.LocalUploadDir, cfg.LocalPublicURL)
}

func providePermissionCache(rdb *redis.Client, cfg *Config) cache.Cache {
	if cfg.PermissionTreeRedisEnable {
		if rdb != nil {
			return cache.NewRedisCache(rdb)
		}
		zap.L().Warn("permission tree redis cache enabled but redis client is nil, fallback to memory")
	}
	return cache.NewMemoryCache(5 * time.Minute)
}

func provideFileOptions(cfg *Config) service.FileOptions {
	return service.FileOptions{
		MaxFileSizeBytes:  int64(cfg.MaxFileSizeMB) << 20,
		MaxChunkSizeBytes: int64(cfg.ChunkSizeMB) << 20,
	}
}

func provideFileService(fileRepo *repository.FileRepository, opts service.FileOptions) *service.FileService {
	return service.NewFileService(fileRepo, opts)
}

func provideAuthService(userRepo *repository.UserRepository, permRepo *repository.PermissionRepository, rdb *redis.Client, jwtManager *jwt.Manager, avatarStorage *storage.LocalStorage) *service.AuthService {
	return service.NewAuthService(userRepo, permRepo, rdb, jwtManager, avatarStorage)
}

func provideBrandService(brandRepo *repository.BrandConfigRepository, avatarStorage *storage.LocalStorage) *service.BrandService {
	return service.NewBrandServiceWithStorage(brandRepo, avatarStorage)
}

func providePermissionService(permRepo *repository.PermissionRepository, permCache cache.Cache) *service.PermissionService {
	return service.NewPermissionService(permRepo, permCache)
}

func provideWSHandler(hub *ws.Hub, jwtManager *jwt.Manager, cfg *Config) *handler.WSHandler {
	return handler.NewWSHandler(hub, jwtManager, cfg.AllowOrigins)
}

func provideStreamManager(deviceRepo *repository.DeviceRepository, mediaStreamRepo *repository.MediaStreamRepository) *service.StreamManager {
	engineClient := &service.MockEngineClient{}
	sm := service.NewStreamManager(engineClient, deviceRepo, mediaStreamRepo, zap.L())
	sm.StartBackgroundTasks(context.Background())
	return sm
}

func provideDeviceStagingHandler(
	stagingSvc *service.DeviceStagingService,
	discoverySvc *service.DeviceDiscoveryService,
) *handler.DeviceStagingHandler {
	return handler.NewDeviceStagingHandler(stagingSvc, discoverySvc)
}

func provideDeviceHandler(
	deviceRepo *repository.DeviceRepository,
	permCache cache.Cache,
	taskClient *task.Client,
	zlmClient *zlm.Client,
	streamManager *service.StreamManager,
) *handler.DeviceHandler {
	deviceSvc := service.NewDeviceService(deviceRepo, permCache, taskClient, zlmClient, streamManager)
	return handler.NewDeviceHandler(deviceSvc)
}

func provideDeviceGroupHandler(groupRepo *repository.DeviceGroupRepository) *handler.DeviceGroupHandler {
	groupSvc := service.NewDeviceGroupService(groupRepo)
	return handler.NewDeviceGroupHandler(groupSvc)
}

func provideDeviceStagingService(
	stagingRepo *repository.DiscoveredDeviceRepository,
	deviceRepo *repository.DeviceRepository,
) *service.DeviceStagingService {
	return service.NewDeviceStagingService(stagingRepo, deviceRepo)
}

func provideDeviceDiscoveryService(stagingSvc *service.DeviceStagingService) *service.DeviceDiscoveryService {
	onvifScanner := onvif.NewScanner()
	return service.NewDeviceDiscoveryService(stagingSvc, onvifScanner, nil)
}

func provideSystemHandler(db *gorm.DB, rdb *redis.Client, cfg *Config, scheduler *asynq.Scheduler) *handler.SystemHandler {
	systemSvc := service.NewSystemService(db, rdb, buildinfo.Version, cfg.ZLMAPIURL, "")
	systemInfoSvc := service.NewSystemInfoService(db)
	networkSvc := service.NewNetworkService(db)
	timeConfigSvc := service.NewTimeConfigService(db)
	webhookSvc := service.NewWebhookService(db)
	storageSvc := service.NewStorageService(db, rdb)

	var storageScheduler *task.StorageScheduler
	if scheduler != nil {
		storageScheduler = task.NewStorageScheduler(scheduler, storageSvc)
		if err := storageScheduler.Init(); err != nil {
			zap.L().Error("failed to init storage scheduler", zap.Error(err))
		}
	}

	return handler.NewSystemHandler(systemSvc, systemInfoSvc, networkSvc, timeConfigSvc, webhookSvc, storageSvc, storageScheduler)
}

func newRouteDeps(
	permCache cache.Cache,
	auditSvc *service.AuditService,
	authHandler *handler.AuthHandler,
	wsHandler *handler.WSHandler,
	userHandler *handler.UserHandler,
	roleHandler *handler.RoleHandler,
	permHandler *handler.PermissionHandler,
	fileHandler *handler.FileHandler,
	auditHandler *handler.AuditHandler,
	taskHandler *handler.TaskHandler,
	brandHandler *handler.BrandHandler,
	mailHandler *handler.MailHandler,
	feedbackHandler *handler.FeedbackHandler,
	dashboardHandler *handler.DashboardHandler,
	deviceHandler *handler.DeviceHandler,
	deviceGroupHandler *handler.DeviceGroupHandler,
	deviceStagingHandler *handler.DeviceStagingHandler,
	systemHandler *handler.SystemHandler,
	streamManager *service.StreamManager,
	licenseHandler *handler.LicenseHandler,
	licenseService *service.LicenseService,
	gb28181Handler *handler.GB28181Handler,
	mediaGB28181Handler *handler.MediaGB28181Handler,
	smartRecordHandler *handler.SmartRecordHandler,
	gb28181ConfigHandler *handler.GB28181ConfigHandler,
	sipService *service.SIPService,
) *RouteDeps {
	return &RouteDeps{
		RBACCache:            permCache,
		AuditService:         auditSvc,
		AuthHandler:          authHandler,
		WSHandler:            wsHandler,
		UserHandler:          userHandler,
		RoleHandler:          roleHandler,
		PermissionHandler:    permHandler,
		FileHandler:          fileHandler,
		AuditHandler:         auditHandler,
		TaskHandler:          taskHandler,
		BrandHandler:         brandHandler,
		MailHandler:          mailHandler,
		FeedbackHandler:      feedbackHandler,
		DashboardHandler:     dashboardHandler,
		DeviceHandler:        deviceHandler,
		DeviceGroupHandler:   deviceGroupHandler,
		DeviceStagingHandler: deviceStagingHandler,
		SystemHandler:        systemHandler,
		StreamManager:        streamManager,
		LicenseHandler:       licenseHandler,
		LicenseService:       licenseService,
		GB28181Handler:       gb28181Handler,
		MediaGB28181Handler:  mediaGB28181Handler,
		SmartRecordHandler:   smartRecordHandler,
		GB28181ConfigHandler: gb28181ConfigHandler,
		SIPService:           sipService,
	}
}

func provideMailServiceForAsynq(db *gorm.DB) *service.MailService {
	mailConfigRepo := repository.NewMailConfigRepository(db)
	inboundEmailRepo := repository.NewInboundEmailRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	return service.NewMailService(mailConfigRepo, inboundEmailRepo, feedbackRepo)
}

func provideLicenseHandler(licenseSvc *service.LicenseService) *handler.LicenseHandler {
	return handler.NewLicenseHandler(licenseSvc)
}

func provideZLMClient(cfg *Config) *zlm.Client {
	return zlm.NewClient(cfg.ZLMAPIURL, cfg.ZLMSecret, zap.L())
}

func provideSIPServiceWithZLM(
	deviceRepo *repository.DeviceRepository,
	gbDeviceRepo *repository.GB28181DeviceRepository,
	mediaStreamRepo *repository.MediaStreamRepository,
	smartRecordRepo *repository.SmartRecordRepository,
	deviceSipConfigRepo *repository.DeviceSipConfigRepository,
	deviceRepoV2 *repository.DeviceRepositoryV2,
	taskClient *task.Client,
	zlmClient *zlm.Client,
	streamManager *service.StreamManager,
	cfg *Config,
	cache cache.Cache,
	hub *ws.Hub,
) *service.SIPService {
	zlmBaseIP := cfg.ZLMExternalIP
	if zlmBaseIP == "" {
		// 从 ZLMAPIURL 提取 host
		zlmBaseIP = cfg.ZLMAPIURL
	}
	rtmpPort := cfg.ZLMRTMPPort
	if rtmpPort == 0 {
		rtmpPort = 1935
	}
	rtspPort := cfg.ZLMRTSPPort
	if rtspPort == 0 {
		rtspPort = 554
	}
	httpPort := cfg.ZLMHTTPPort
	if httpPort == 0 {
		httpPort = 80
	}
	return service.NewSIPServiceWithZLM(
		deviceRepo, gbDeviceRepo, mediaStreamRepo,
		smartRecordRepo, deviceSipConfigRepo, deviceRepoV2, taskClient,
		zlmClient, streamManager, zlmBaseIP,
		rtmpPort, rtspPort, httpPort,
		cache, hub,
	)
}

func provideMediaServices(db *gorm.DB, cfg *Config, streamManager *service.StreamManager, sipSvc *service.SIPService) (*handler.MediaWebhookHandler, *handler.MediaPlayHandler, *handler.MediaRecordingHandler, *handler.DeviceStagingHandler) {
	zlmClient := provideZLMClient(cfg)
	deviceRepo := repository.NewDeviceRepository(db)
	mediaStreamRepo := repository.NewMediaStreamRepository(db)
	recordingRepo := repository.NewRecordingRepository(db)
	stagingRepo := repository.NewDiscoveredDeviceRepository(db)

	onvifScanner := onvif.NewScanner()
	stagingSvc := service.NewDeviceStagingService(stagingRepo, deviceRepo)
	discoverySvc := service.NewDeviceDiscoveryService(stagingSvc, onvifScanner, nil)

	mediaSvc := service.NewMediaService(zlmClient, mediaStreamRepo, deviceRepo, streamManager, cfg.ZLMAPIURL, cfg.ZLMSecret)
	recordingSvc := service.NewRecordingService(recordingRepo, zlmClient)

	webhookHandler := handler.NewMediaWebhookHandler(mediaSvc, sipSvc, streamManager, stagingSvc)
	playHandler := handler.NewMediaPlayHandler(mediaSvc)
	recordingHandler := handler.NewMediaRecordingHandler(recordingSvc)
	stagingHandler := handler.NewDeviceStagingHandler(stagingSvc, discoverySvc)

	return webhookHandler, playHandler, recordingHandler, stagingHandler
}

func provideGB28181Handler(sipSvc *service.SIPService, c cache.Cache, hub *ws.Hub) *handler.GB28181Handler {
	return handler.NewGB28181Handler(sipSvc, c, hub)
}

func provideMediaGB28181Handler(sipSvc *service.SIPService) *handler.MediaGB28181Handler {
	return handler.NewMediaGB28181Handler(sipSvc)
}

func provideSmartRecordHandler(repo *repository.SmartRecordRepository) *handler.SmartRecordHandler {
	return handler.NewSmartRecordHandler(repo)
}

func provideGB28181ConfigHandler(zlmClient *zlm.Client) *handler.GB28181ConfigHandler {
	return handler.NewGB28181ConfigHandler(zlmClient)
}
