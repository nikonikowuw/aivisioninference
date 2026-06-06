package router

import (
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/buildinfo"
	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

// RouteDeps 聚合路由注册阶段需要的 Handler、Service 与缓存依赖。
type RouteDeps struct {
	RBACCache          cache.Cache
	AuditService       *service.AuditService
	AuthHandler        *handler.AuthHandler
	WSHandler          *handler.WSHandler
	UserHandler        *handler.UserHandler
	RoleHandler        *handler.RoleHandler
	PermissionHandler  *handler.PermissionHandler
	FileHandler        *handler.FileHandler
	AuditHandler       *handler.AuditHandler
	TaskHandler        *handler.TaskHandler
	BrandHandler       *handler.BrandHandler
	MailHandler        *handler.MailHandler
	FeedbackHandler    *handler.FeedbackHandler
	DashboardHandler   *handler.DashboardHandler
	DeviceHandler      *handler.DeviceHandler
	DeviceGroupHandler *handler.DeviceGroupHandler
	SystemHandler      *handler.SystemHandler
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

func provideDeviceHandler(
	deviceRepo *repository.DeviceRepository,
	permCache cache.Cache,
	taskClient *task.Client,
	zlmClient *zlm.Client,
) *handler.DeviceHandler {
	deviceSvc := service.NewDeviceService(deviceRepo, permCache, taskClient, zlmClient)
	return handler.NewDeviceHandler(deviceSvc)
}

func provideDeviceGroupHandler(groupRepo *repository.DeviceGroupRepository) *handler.DeviceGroupHandler {
	groupSvc := service.NewDeviceGroupService(groupRepo)
	return handler.NewDeviceGroupHandler(groupSvc)
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
	systemHandler *handler.SystemHandler,
) *RouteDeps {
	return &RouteDeps{
		RBACCache:         permCache,
		AuditService:      auditSvc,
		AuthHandler:       authHandler,
		WSHandler:         wsHandler,
		UserHandler:       userHandler,
		RoleHandler:       roleHandler,
		PermissionHandler: permHandler,
		FileHandler:       fileHandler,
		AuditHandler:      auditHandler,
		TaskHandler:       taskHandler,
		BrandHandler:      brandHandler,
		MailHandler:       mailHandler,
		FeedbackHandler:   feedbackHandler,
		DashboardHandler:   dashboardHandler,
		DeviceHandler:      deviceHandler,
		DeviceGroupHandler: deviceGroupHandler,
		SystemHandler:      systemHandler,
	}
}

func provideMailServiceForAsynq(db *gorm.DB) *service.MailService {
	mailConfigRepo := repository.NewMailConfigRepository(db)
	inboundEmailRepo := repository.NewInboundEmailRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	return service.NewMailService(mailConfigRepo, inboundEmailRepo, feedbackRepo)
}

func provideZLMClient(cfg *Config) *zlm.Client {
	return zlm.NewClient(cfg.ZLMAPIURL, cfg.ZLMSecret, zap.L())
}

func provideMediaServices(db *gorm.DB, cfg *Config) (*handler.MediaWebhookHandler, *handler.MediaPlayHandler, *handler.MediaRecordingHandler) {
	zlmClient := provideZLMClient(cfg)
	deviceRepo := repository.NewDeviceRepository(db)
	mediaStreamRepo := repository.NewMediaStreamRepository(db)
	gbDeviceRepo := repository.NewGB28181DeviceRepository(db)
	recordingRepo := repository.NewRecordingRepository(db)

	sipSvc := service.NewSIPService(deviceRepo, gbDeviceRepo, mediaStreamRepo)
	mediaSvc := service.NewMediaService(zlmClient, mediaStreamRepo, deviceRepo, cfg.ZLMAPIURL, cfg.ZLMSecret)
	recordingSvc := service.NewRecordingService(recordingRepo, zlmClient)

	webhookHandler := handler.NewMediaWebhookHandler(mediaSvc, sipSvc)
	playHandler := handler.NewMediaPlayHandler(mediaSvc)
	recordingHandler := handler.NewMediaRecordingHandler(recordingSvc)

	return webhookHandler, playHandler, recordingHandler
}
