//go:build wireinject
// +build wireinject

package router

import (
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
	repository.NewLicenseRepository,
	repository.NewAIVisionTaskRepository,
	repository.NewAITimeScheduleRepository,
	repository.NewGB28181DeviceRepository,
)

var serviceSet = wire.NewSet(
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
	task.NewClient,
	provideZLMClient,
	provideStreamManager,
	provideDeviceStagingService,
	provideDeviceDiscoveryService,
	provideSystemHandler,
	provideLicenseService,
	provideAIVisionTaskService,
	provideSIPServiceWithZLM,
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
	provideDeviceStagingHandler,
	provideDeviceHandler,
	provideDeviceGroupHandler,
	provideLicenseHandler,
	provideAIVisionTaskHandler,
	provideAITimeScheduleHandler,
)

// InitializeRouteDeps 使用 Wire 构造路由注册所需依赖。
func InitializeRouteDeps(db *gorm.DB, rdb *redis.Client, jwtManager *jwt.Manager, hub *ws.Hub, cfg *Config, scheduler *asynq.Scheduler) (*RouteDeps, error) {
	wire.Build(
		repositorySet,
		serviceSet,
		handlerSet,
		newRouteDeps,
	)
	return nil, nil
}
