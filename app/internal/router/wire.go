//go:build wireinject
// +build wireinject

package router

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/wire"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
)

var repositorySet = wire.NewSet(
	repository.NewUserRepository,
	repository.NewRoleRepository,
	repository.NewPermissionRepository,
	repository.NewAuditRepository,
	repository.NewFileRepository,
	repository.NewTaskRepository,
	repository.NewDashboardRepository,
	repository.NewBrandConfigRepository,
	repository.NewMailConfigRepository,
	repository.NewEmailTokenRepository,
	repository.NewInboundEmailRepository,
	repository.NewFeedbackRepository,
	repository.NewDeviceRepository,
	repository.NewDeviceGroupRepository,
	repository.NewMediaStreamRepository,
	repository.NewDiscoveredDeviceRepository,
	repository.NewSmartRecordRepository,
	repository.NewLicenseRepository,
	repository.NewAIVisionTaskRepository,
	repository.NewAITimeScheduleRepository,
	repository.NewGB28181DeviceRepository,
	repository.NewAlgorithmPackageRepository,
	repository.NewPersonRepository,
	repository.NewPersonGroupRepository,
	repository.NewPersonTagRepository,
	repository.NewPersonTagRelationRepository,
	repository.NewPersonEmbeddingRepository,
	repository.NewImportTaskRepository,
	repository.NewDeviceSipConfigRepository,
	repository.NewDeviceRepositoryV2,
	repository.NewGB28181PlatformConfigRepository,
	repository.NewGB28181StreamSessionRepository,
	repository.NewEdgeNodeRepository,
	repository.NewEdgeNodeAlgorithmRepository,
	repository.NewEdgeNodeTagRepository,
	repository.NewEdgeScheduledTaskRepository,
	repository.NewEdgeScheduledTaskRecordRepository,
	repository.NewTerminalSessionRepository,
)

var serviceSet = wire.NewSet(
	provideFileStorage,
	provideAvatarStorage,
	providePermissionCache,
	provideFileOptions,
	provideFileService,
	provideAuthService,
	provideBrandService,
	providePermissionService,
	service.NewAuditService,
	wire.Bind(new(middleware.AuditLogger), new(*service.AuditService)),
	service.NewUserService,
	service.NewRoleService,
	service.NewTaskService,
	service.NewDashboardService,
	service.NewMailService,
	service.NewEmailVerificationService,
	service.NewFeedbackService,
	service.NewSmartRecordService,
	task.NewClient,
	provideZLMClient,
	provideStreamManager,
	provideDeviceStagingService,
	provideDeviceDiscoveryService,
	provideSystemHandler,
	provideLicenseService,
	service.NewAIVisionTaskService,
	provideSIPServiceWithZLM,
	provideEngineClient,
	provideAlgorithmOptions,
	provideAlgorithmPackageService,
	provideGB28181PlatformConfigService,
	service.NewSIPRuntimeService,
	provideEdgeNodeService,
	provideEdgeNodeTagService,
	provideEdgeScheduledTaskService,
	provideMqttSyncManager,
	provideMqttMux,
	provideHistoryBuffer,
	provideEngineMetricsStore,
	provideEdgeMqttHandler,
	service.NewSSHPool,
	provideTerminalHandler,
)

var handlerSet = wire.NewSet(
	handler.NewAuthHandlerWithEmail,
	provideWSHandler,
	handler.NewUserHandler,
	handler.NewRoleHandler,
	handler.NewPermissionHandler,
	handler.NewFileHandler,
	handler.NewAuditHandler,
	handler.NewTaskHandler,
	handler.NewBrandHandler,
	handler.NewMailHandler,
	handler.NewFeedbackHandler,
	handler.NewDashboardHandler,
	handler.NewSmartRecordHandler,
	provideDeviceStagingHandler,
	provideDeviceHandler,
	provideDeviceGroupHandler,
	provideLicenseHandler,
	provideAIVisionTaskHandler,
	provideAITimeScheduleHandler,
	handler.NewAlgorithmPackageHandler,
	providePersonHandler,
	provideGB28181Handler,
	provideMediaGB28181Handler,
	provideGB28181ConfigHandler,
	provideEdgeNodeHandler,
	provideEdgeNodeTagHandler,
	provideEdgeScheduledTaskHandler,
	provideEdgeNodeMiddleware,
)

// InitializeRouteDeps 使用 Wire 构造路由注册所需依赖。
func InitializeRouteDeps(db *gorm.DB, rdb *redis.Client, jwtManager *jwt.Manager, hub *ws.Hub, cfg *Config, scheduler *asynq.Scheduler, mqttClient mqtt.Client) (*RouteDeps, error) {
	wire.Build(
		repositorySet,
		serviceSet,
		handlerSet,
		newRouteDeps,
	)
	return nil, nil
}
