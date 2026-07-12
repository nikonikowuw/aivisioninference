package router

import (
	"context"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/buildinfo"
	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttmux"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttsync"
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
	RBACCache               cache.Cache
	AuditService            *service.AuditService
	AuthHandler             *handler.AuthHandler
	WSHandler               *handler.WSHandler
	UserHandler             *handler.UserHandler
	RoleHandler             *handler.RoleHandler
	PermissionHandler       *handler.PermissionHandler
	FileHandler             *handler.FileHandler
	AuditHandler            *handler.AuditHandler
	TaskHandler             *handler.TaskHandler
	BrandHandler            *handler.BrandHandler
	MailHandler             *handler.MailHandler
	FeedbackHandler         *handler.FeedbackHandler
	DashboardHandler        *handler.DashboardHandler
	DeviceHandler           *handler.DeviceHandler
	DeviceGroupHandler      *handler.DeviceGroupHandler
	DeviceStagingHandler    *handler.DeviceStagingHandler
	SystemHandler           *handler.SystemHandler
	SmartRecordHandler      *handler.SmartRecordHandler
	StreamManager           *service.StreamManager
	LicenseHandler          *handler.LicenseHandler
	LicenseService          *service.LicenseService
	AIVisionTaskHandler     *handler.AIVisionTaskHandler
	AITimeScheduleHandler   *handler.AITimeScheduleHandler
	AlgorithmPackageHandler *handler.AlgorithmPackageHandler
	PersonHandler           *handler.PersonHandler
	GB28181Handler          *handler.GB28181Handler
	MediaGB28181Handler     *handler.MediaGB28181Handler
	GB28181ConfigHandler    *handler.GB28181ConfigHandler
	SIPService              *service.SIPService
	SIPRuntimeSvc           *service.SIPRuntimeService
	EdgeNodeHandler         *handler.EdgeNodeHandler
	EdgeNodeMiddleware      *middleware.EdgeNodeMiddleware
	EdgeNodeSvc             *service.EdgeNodeService
	EdgeMqttHandler         *handler.EdgeMqttHandler
	MqttMux                 *mqttmux.Mux
	EngineMetricsStore      *service.EngineMetricsStore
	EdgeNodeMetricsHandler  *handler.EdgeNodeMetricsHandler

	// Phase 2: Alert Engine.
	AlertRuleRepo         *repository.AlertRuleRepository
	AlertEventRepo        *repository.AlertEventRepository
	AlertRuleService      *service.AlertRuleService
	AlertEventService     *service.AlertEventService
	AlertRuleHandler      *handler.AlertRuleHandler
	AlertEventHandler     *handler.AlertEventHandler
	AlertEngine           *service.AlertEngine

	// Phase 3: Remote Operations.
	EdgeNodeScheduledTaskRepo       *repository.EdgeNodeScheduledTaskRepository
	EdgeNodeTaskExecutionRepo       *repository.EdgeNodeTaskExecutionRepository
	EdgeNodeScheduledTaskService    *service.EdgeNodeScheduledTaskService
	EdgeNodeTerminalService         *service.EdgeNodeTerminalService
	EdgeNodeScheduledTaskHandler    *handler.EdgeNodeScheduledTaskHandler
	EdgeNodeTerminalHandler         *handler.EdgeNodeTerminalHandler
}

func provideFileStorage(cfg *Config) (storage.Storage, error) {
	// 根据配置中的 StorageType 一键切换后端
	switch strings.ToLower(cfg.StorageType) {
	case "oss", "minio", "s3":
		zap.L().Info("using OSS/MinIO as file storage", zap.String("endpoint", cfg.OSSEndpoint), zap.String("bucket", cfg.OSSBucket))
		return storage.NewOSSStorage(
			cfg.OSSEndpoint,
			cfg.OSSAccessKey,
			cfg.OSSSecretKey,
			cfg.OSSBucket,
			cfg.OSSUseSSL,
		)
	default:
		zap.L().Info("using local filesystem as file storage", zap.String("dir", cfg.LocalUploadDir))
		return storage.NewLocalStorage(cfg.LocalUploadDir, cfg.LocalPublicURL)
	}
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

func provideStreamManager(engineClient service.EngineClient, deviceRepo *repository.DeviceRepository, mediaStreamRepo *repository.MediaStreamRepository) *service.StreamManager {
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
	discoveredDeviceRepo *repository.DiscoveredDeviceRepository,
	permCache cache.Cache,
	taskClient *task.Client,
	zlmClient *zlm.Client,
	streamManager *service.StreamManager,
) *handler.DeviceHandler {
	deviceSvc := service.NewDeviceService(deviceRepo, discoveredDeviceRepo, permCache, taskClient, zlmClient, streamManager)
	return handler.NewDeviceHandler(deviceSvc)
}

func provideDeviceGroupHandler(groupRepo *repository.DeviceGroupRepository) *handler.DeviceGroupHandler {
	groupSvc := service.NewDeviceGroupService(groupRepo)
	return handler.NewDeviceGroupHandler(groupSvc)
}

func provideDeviceStagingService(
	stagingRepo *repository.DiscoveredDeviceRepository,
	deviceRepo *repository.DeviceRepository,
	gbDeviceRepo *repository.GB28181DeviceRepository,
	deviceSipConfigRepo *repository.DeviceSipConfigRepository,
) *service.DeviceStagingService {
	return service.NewDeviceStagingService(stagingRepo, deviceRepo, gbDeviceRepo, deviceSipConfigRepo)
}

func provideDeviceDiscoveryService(stagingSvc *service.DeviceStagingService) *service.DeviceDiscoveryService {
	onvifScanner := onvif.NewScanner()
	return service.NewDeviceDiscoveryService(stagingSvc, onvifScanner, nil)
}

func provideSystemHandler(db *gorm.DB, rdb *redis.Client, cfg *Config, scheduler *asynq.Scheduler, metricsStore *service.EngineMetricsStore) *handler.SystemHandler {
	systemSvc := service.NewSystemService(db, rdb, buildinfo.Version, cfg.ZLMAPIURL, "")
	systemSvc.SetEngineMetricsStore(metricsStore)
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

func provideHistoryBuffer() *service.HistoryBuffer {
	return service.NewHistoryBuffer(60)
}

func provideEngineMetricsStore(history *service.HistoryBuffer) *service.EngineMetricsStore {
	return service.NewEngineMetricsStore(history)
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
	smartRecordHandler *handler.SmartRecordHandler,
	streamManager *service.StreamManager,
	licenseHandler *handler.LicenseHandler,
	licenseService *service.LicenseService,
	aiVisionTaskHandler *handler.AIVisionTaskHandler,
	aiTimeScheduleHandler *handler.AITimeScheduleHandler,
	algorithmPackageHandler *handler.AlgorithmPackageHandler,
	personHandler *handler.PersonHandler,
	gb28181Handler *handler.GB28181Handler,
	mediaGB28181Handler *handler.MediaGB28181Handler,
	gb28181ConfigHandler *handler.GB28181ConfigHandler,
	sipService *service.SIPService,
	sipRuntimeSvc *service.SIPRuntimeService,
	edgeNodeHandler *handler.EdgeNodeHandler,
	edgeNodeMiddleware *middleware.EdgeNodeMiddleware,
	edgeMqttHandler *handler.EdgeMqttHandler,
	mqttMux *mqttmux.Mux,
	metricsStore *service.EngineMetricsStore,
	edgeNodeMetricsHandler *handler.EdgeNodeMetricsHandler,
	alertRuleHandler *handler.AlertRuleHandler,
	alertEventHandler *handler.AlertEventHandler,
	alertEngine *service.AlertEngine,

	// Phase 3: Remote Operations.
	edgeNodeScheduledTaskRepo *repository.EdgeNodeScheduledTaskRepository,
	edgeNodeTaskExecutionRepo *repository.EdgeNodeTaskExecutionRepository,
	edgeNodeScheduledTaskService *service.EdgeNodeScheduledTaskService,
	edgeNodeTerminalService *service.EdgeNodeTerminalService,
	edgeNodeScheduledTaskHandler *handler.EdgeNodeScheduledTaskHandler,
	edgeNodeTerminalHandler *handler.EdgeNodeTerminalHandler,
) *RouteDeps {
	if sipService != nil {
		sipService.SetRuntimeService(sipRuntimeSvc)
	}
	return &RouteDeps{
		RBACCache:               permCache,
		AuditService:            auditSvc,
		AuthHandler:             authHandler,
		WSHandler:               wsHandler,
		UserHandler:             userHandler,
		RoleHandler:             roleHandler,
		PermissionHandler:       permHandler,
		FileHandler:             fileHandler,
		AuditHandler:            auditHandler,
		TaskHandler:             taskHandler,
		BrandHandler:            brandHandler,
		MailHandler:             mailHandler,
		FeedbackHandler:         feedbackHandler,
		DashboardHandler:        dashboardHandler,
		DeviceHandler:           deviceHandler,
		DeviceGroupHandler:      deviceGroupHandler,
		DeviceStagingHandler:    deviceStagingHandler,
		SystemHandler:           systemHandler,
		SmartRecordHandler:      smartRecordHandler,
		StreamManager:           streamManager,
		LicenseHandler:          licenseHandler,
		LicenseService:          licenseService,
		AIVisionTaskHandler:     aiVisionTaskHandler,
		AITimeScheduleHandler:   aiTimeScheduleHandler,
		AlgorithmPackageHandler: algorithmPackageHandler,
		PersonHandler:           personHandler,
		GB28181Handler:          gb28181Handler,
		MediaGB28181Handler:     mediaGB28181Handler,
		GB28181ConfigHandler:    gb28181ConfigHandler,
		SIPService:              sipService,
		SIPRuntimeSvc:           sipRuntimeSvc,
		EdgeNodeHandler:         edgeNodeHandler,
		EdgeNodeMiddleware:      edgeNodeMiddleware,
		EdgeMqttHandler:         edgeMqttHandler,
		MqttMux:                 mqttMux,
		EngineMetricsStore:      metricsStore,
		EdgeNodeMetricsHandler:  edgeNodeMetricsHandler,
		AlertRuleHandler:        alertRuleHandler,
		AlertEventHandler:       alertEventHandler,
		AlertEngine:             alertEngine,

		// Phase 3: Remote Operations.
		EdgeNodeScheduledTaskRepo:       edgeNodeScheduledTaskRepo,
		EdgeNodeTaskExecutionRepo:       edgeNodeTaskExecutionRepo,
		EdgeNodeScheduledTaskService:    edgeNodeScheduledTaskService,
		EdgeNodeTerminalService:         edgeNodeTerminalService,
		EdgeNodeScheduledTaskHandler:    edgeNodeScheduledTaskHandler,
		EdgeNodeTerminalHandler:         edgeNodeTerminalHandler,
	}
}

func provideMailServiceForAsynq(db *gorm.DB) *service.MailService {
	mailConfigRepo := repository.NewMailConfigRepository(db)
	inboundEmailRepo := repository.NewInboundEmailRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	return service.NewMailService(mailConfigRepo, inboundEmailRepo, feedbackRepo)
}

func provideLicenseHandler(licenseRepo *repository.LicenseRepository, db *gorm.DB) *handler.LicenseHandler {
	licenseSvc := service.NewLicenseService(licenseRepo, db)
	return handler.NewLicenseHandler(licenseSvc)
}

func provideLicenseService(licenseRepo *repository.LicenseRepository, db *gorm.DB) *service.LicenseService {
	return service.NewLicenseService(licenseRepo, db)
}

func provideAIVisionTaskHandler(repo *repository.AIVisionTaskRepository, scheduleRepo *repository.AITimeScheduleRepository, algorithmPackageRepo *repository.AlgorithmPackageRepository, deviceRepo *repository.DeviceRepository, nodeRepo *repository.EdgeNodeRepository, nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository, sipSvc *service.SIPService, streamManager *service.StreamManager) *handler.AIVisionTaskHandler {
	policy := service.NewInferenceNodePolicy(nodeRepo, nodeAlgoRepo)
	svc := service.NewAIVisionTaskService(repo, scheduleRepo, algorithmPackageRepo, deviceRepo, sipSvc, streamManager, policy)
	return handler.NewAIVisionTaskHandler(svc)
}

func provideAITimeScheduleHandler(repo *repository.AITimeScheduleRepository) *handler.AITimeScheduleHandler {
	svc := service.NewAITimeScheduleService(repo)
	return handler.NewAITimeScheduleHandler(svc)
}

func provideZLMClient(cfg *Config) *zlm.Client {
	return zlm.NewClient(cfg.ZLMAPIURL, cfg.ZLMSecret, zap.L())
}

// defaultPort 返回端口值，若为 0 则返回默认值。
func defaultPort(val, defaultVal int) int {
	if val == 0 {
		return defaultVal
	}
	return val
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
	auditRepo *repository.AuditRepository,
	streamSessionRepo *repository.GB28181StreamSessionRepository,
) *service.SIPService {
	zlmBaseIP := cfg.ZLMExternalIP
	if zlmBaseIP == "" {
		zlmBaseIP = cfg.ZLMAPIURL
	}
	rtmpPort := defaultPort(cfg.ZLMRTMPPort, 1935)
	rtspPort := defaultPort(cfg.ZLMRTSPPort, 554)
	httpPort := defaultPort(cfg.ZLMHTTPPort, 80)
	return service.NewSIPServiceWithZLM(
		deviceRepo, gbDeviceRepo, mediaStreamRepo,
		smartRecordRepo, deviceSipConfigRepo, deviceRepoV2, taskClient,
		zlmClient, streamManager, zlmBaseIP,
		rtmpPort, rtspPort, httpPort,
		cache, hub,
		auditRepo,
		streamSessionRepo,
	)
}

func provideMediaServices(db *gorm.DB, cfg *Config, streamManager *service.StreamManager, sipSvc *service.SIPService) (*handler.MediaWebhookHandler, *handler.MediaPlayHandler, *handler.MediaRecordingHandler, *handler.DeviceStagingHandler) {
	zlmClient := provideZLMClient(cfg)
	deviceRepo := repository.NewDeviceRepository(db)
	mediaStreamRepo := repository.NewMediaStreamRepository(db)
	recordingRepo := repository.NewRecordingRepository(db)
	stagingRepo := repository.NewDiscoveredDeviceRepository(db)

	gbDeviceRepo := repository.NewGB28181DeviceRepository(db)
	deviceSipConfigRepo := repository.NewDeviceSipConfigRepository(db)

	onvifScanner := onvif.NewScanner()
	stagingSvc := service.NewDeviceStagingService(stagingRepo, deviceRepo, gbDeviceRepo, deviceSipConfigRepo)
	discoverySvc := service.NewDeviceDiscoveryService(stagingSvc, onvifScanner, nil)

	mediaSvc := service.NewMediaService(zlmClient, mediaStreamRepo, deviceRepo, streamManager, cfg.ZLMAPIURL, cfg.ZLMSecret)
	recordingSvc := service.NewRecordingService(recordingRepo, zlmClient)

	webhookHandler := handler.NewMediaWebhookHandler(mediaSvc, sipSvc, streamManager, stagingSvc)
	playHandler := handler.NewMediaPlayHandler(mediaSvc)
	recordingHandler := handler.NewMediaRecordingHandler(recordingSvc)
	stagingHandler := handler.NewDeviceStagingHandler(stagingSvc, discoverySvc)

	return webhookHandler, playHandler, recordingHandler, stagingHandler
}

func provideEngineClient(mqttClient mqtt.Client, syncManager *mqttsync.MqttSyncManager) service.EngineClient {
	return service.NewMqttEngineClient(mqttClient, syncManager, zap.L())
}

func provideAlgorithmPackageService(
	algorithmpackageRepo *repository.AlgorithmPackageRepository,
	opts service.AlgorithmOptions,
	rdb *redis.Client,
	engine service.EngineClient,
	taskClient *task.Client,
) *service.AlgorithmPackageService {
	return service.NewAlgorithmPackageService(algorithmpackageRepo, opts, rdb, engine, taskClient)
}

func provideAlgorithmOptions(cfg *Config) service.AlgorithmOptions {
	return service.AlgorithmOptions{
		MaxAlgoFileSizeBytes: int64(cfg.MaxAlgoFileSizeMB) << 20,
		LocalUploadDir:       cfg.LocalUploadDir,
		PublicURL:            cfg.LocalPublicURL,
	}
}

func providePersonHandler(
	personRepo *repository.PersonRepository,
	groupRepo *repository.PersonGroupRepository,
	tagRepo *repository.PersonTagRepository,
	tagRelationRepo *repository.PersonTagRelationRepository,
	importTaskRepo *repository.ImportTaskRepository,
	fileStorage storage.Storage,
	taskClient *task.Client,
	embeddingRepo *repository.PersonEmbeddingRepository,
	algorithmPackageRepo *repository.AlgorithmPackageRepository,
	nodeRepo *repository.EdgeNodeRepository,
	rdb *redis.Client,
	engine service.EngineClient,
) *handler.PersonHandler {
	embeddingScheduler := service.NewFaceEmbeddingScheduler(nodeRepo, rdb)
	personSvc := service.NewPersonService(service.PersonServiceDeps{
		PersonRepo:           personRepo,
		GroupRepo:            groupRepo,
		TagRepo:              tagRepo,
		TagRelationRepo:      tagRelationRepo,
		ImportTaskRepo:       importTaskRepo,
		Storage:              fileStorage,
		TaskClient:           taskClient,
		EmbeddingRepo:        embeddingRepo,
		AlgorithmPackageRepo: algorithmPackageRepo,
		EmbeddingScheduler:   embeddingScheduler,
		Engine:               engine,
	})
	return handler.NewPersonHandler(personSvc)
}

func provideGB28181Handler(sipSvc *service.SIPService, c cache.Cache, hub *ws.Hub) *handler.GB28181Handler {
	return handler.NewGB28181Handler(sipSvc, c, hub)
}

func provideMediaGB28181Handler(sipSvc *service.SIPService) *handler.MediaGB28181Handler {
	return handler.NewMediaGB28181Handler(sipSvc)
}

func provideSmartRecordHandler(svc *service.SmartRecordService) *handler.SmartRecordHandler {
	return handler.NewSmartRecordHandler(svc)
}

func provideGB28181ConfigHandler(zlmClient *zlm.Client, platformConfigSvc *service.GB28181PlatformConfigService) *handler.GB28181ConfigHandler {
	return handler.NewGB28181ConfigHandler(zlmClient, platformConfigSvc)
}

func provideGB28181PlatformConfigService(
	repo *repository.GB28181PlatformConfigRepository,
	runtimeSvc *service.SIPRuntimeService,
) *service.GB28181PlatformConfigService {
	svc := service.NewGB28181PlatformConfigService(repo)
	svc.SetNotifier(runtimeSvc)
	return svc
}

func provideEdgeNodeService(
	nodeRepo *repository.EdgeNodeRepository,
	nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository,
	algoPackageRepo *repository.AlgorithmPackageRepository,
	taskRepo *repository.AIVisionTaskRepository,
	deviceRepo *repository.DeviceRepository,
	smartRecordRepo *repository.SmartRecordRepository,
	jwtManager *jwt.Manager,
	fileStorage storage.Storage,
	cfg *Config,
	hub *ws.Hub,
	streamManager *service.StreamManager,
	metricsRepo *repository.EdgeNodeMetricsRepository,
	alertEngine *service.AlertEngine,
) *service.EdgeNodeService {
	svc := service.NewEdgeNodeService(
		nodeRepo,
		nodeAlgoRepo,
		algoPackageRepo,
		taskRepo,
		deviceRepo,
		smartRecordRepo,
		jwtManager,
		fileStorage,
		hub,
		streamManager,
		metricsRepo,
		alertEngine,
	)
	svc.SetVersionConfig(cfg.Engine.MinCompatibleVersion, cfg.Engine.VersionCheckEnabled)
	return svc
}

func provideEdgeNodeHandler(svc *service.EdgeNodeService) *handler.EdgeNodeHandler {
	return handler.NewEdgeNodeHandler(svc)
}

func provideEdgeNodeMiddleware(jwtManager *jwt.Manager) *middleware.EdgeNodeMiddleware {
	return middleware.NewEdgeNodeMiddleware(jwtManager)
}

func provideEdgeNodeMetricsRepository(db *gorm.DB) *repository.EdgeNodeMetricsRepository {
	return repository.NewEdgeNodeMetricsRepository(db)
}

func provideEdgeNodeMetricsService(metricsRepo *repository.EdgeNodeMetricsRepository) *service.EdgeNodeMetricsService {
	return service.NewEdgeNodeMetricsService(metricsRepo)
}

func provideEdgeNodeMetricsHandler(
	svc *service.EdgeNodeMetricsService,
	edgeNodeSvc *service.EdgeNodeService,
) *handler.EdgeNodeMetricsHandler {
	return handler.NewEdgeNodeMetricsHandler(svc, edgeNodeSvc)
}

func provideAlertRuleRepository(db *gorm.DB) *repository.AlertRuleRepository {
	return repository.NewAlertRuleRepository(db)
}

func provideAlertEventRepository(db *gorm.DB) *repository.AlertEventRepository {
	return repository.NewAlertEventRepository(db)
}

func provideAlertRuleService(ruleRepo *repository.AlertRuleRepository) *service.AlertRuleService {
	return service.NewAlertRuleService(ruleRepo)
}

func provideAlertEventService(eventRepo *repository.AlertEventRepository) *service.AlertEventService {
	return service.NewAlertEventService(eventRepo)
}

func provideAlertRuleHandler(svc *service.AlertRuleService) *handler.AlertRuleHandler {
	return handler.NewAlertRuleHandler(svc)
}

func provideAlertEventHandler(svc *service.AlertEventService) *handler.AlertEventHandler {
	return handler.NewAlertEventHandler(svc)
}

func provideNotifierRegistry() *service.NotifierRegistry {
	// Create empty registry; providers can be configured via config at startup
	return service.NewNotifierRegistry()
}

func provideAlertEngine(
	ruleRepo *repository.AlertRuleRepository,
	eventRepo *repository.AlertEventRepository,
	metricsRepo *repository.EdgeNodeMetricsRepository,
	nodeRepo *repository.EdgeNodeRepository,
	notifier *service.NotifierRegistry,
) *service.AlertEngine {
	engine := service.NewAlertEngine(ruleRepo, eventRepo, metricsRepo, nodeRepo, notifier)
	// Restore silence tracker from database so silence periods survive restarts.
	engine.RestoreSilenceState(context.Background())
	return engine
}

func provideEdgeNodeScheduledTaskRepository(db *gorm.DB) *repository.EdgeNodeScheduledTaskRepository {
	return repository.NewEdgeNodeScheduledTaskRepository(db)
}

func provideEdgeNodeTaskExecutionRepository(db *gorm.DB) *repository.EdgeNodeTaskExecutionRepository {
	return repository.NewEdgeNodeTaskExecutionRepository(db)
}

func provideEdgeNodeScheduledTaskService(
	taskRepo *repository.EdgeNodeScheduledTaskRepository,
	execRepo *repository.EdgeNodeTaskExecutionRepository,
	nodeRepo *repository.EdgeNodeRepository,
	mqttClient mqtt.Client,
	syncManager *mqttsync.MqttSyncManager,
) *service.EdgeNodeScheduledTaskService {
	return service.NewEdgeNodeScheduledTaskService(taskRepo, execRepo, nodeRepo, mqttClient, syncManager)
}

func provideEdgeNodeTerminalService(mqttClient mqtt.Client) *service.EdgeNodeTerminalService {
	return service.NewEdgeNodeTerminalService(mqttClient)
}

func provideEdgeNodeScheduledTaskHandler(svc *service.EdgeNodeScheduledTaskService) *handler.EdgeNodeScheduledTaskHandler {
	return handler.NewEdgeNodeScheduledTaskHandler(svc)
}

func provideEdgeNodeTerminalHandler(terminalSvc *service.EdgeNodeTerminalService) *handler.EdgeNodeTerminalHandler {
	return handler.NewEdgeNodeTerminalHandler(terminalSvc)
}

func provideMqttSyncManager(rdb *redis.Client) *mqttsync.MqttSyncManager {
	return mqttsync.NewMqttSyncManager(rdb)
}

func provideMqttMux() *mqttmux.Mux {
	return mqttmux.NewMux()
}

func provideEdgeMqttHandler(
	nodeSvc *service.EdgeNodeService,
	syncManager *mqttsync.MqttSyncManager,
	taskClient *task.Client,
	rdb *redis.Client,
	hub *ws.Hub,
	metricsStore *service.EngineMetricsStore,
	scheduledTaskSvc *service.EdgeNodeScheduledTaskService,
	terminalSvc *service.EdgeNodeTerminalService,
) *handler.EdgeMqttHandler {
	return handler.NewEdgeMqttHandler(nodeSvc, syncManager, taskClient, rdb, hub, metricsStore, scheduledTaskSvc, terminalSvc)
}
