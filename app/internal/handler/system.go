// Package handler 提供 HTTP 请求处理器
package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/internal/task"
)

// SystemHandler 系统管理 Handler
type SystemHandler struct {
	systemSvc        *service.SystemService
	systemInfoSvc    *service.SystemInfoService
	networkSvc       *service.NetworkService
	timeConfigSvc    *service.TimeConfigService
	webhookSvc       *service.WebhookService
	storageSvc       *service.StorageService
	storageScheduler *task.StorageScheduler
}

// NewSystemHandler 创建系统管理 Handler
func NewSystemHandler(
	systemSvc *service.SystemService,
	systemInfoSvc *service.SystemInfoService,
	networkSvc *service.NetworkService,
	timeConfigSvc *service.TimeConfigService,
	webhookSvc *service.WebhookService,
	storageSvc *service.StorageService,
	storageScheduler *task.StorageScheduler,
) *SystemHandler {
	return &SystemHandler{
		systemSvc:        systemSvc,
		systemInfoSvc:    systemInfoSvc,
		networkSvc:       networkSvc,
		timeConfigSvc:    timeConfigSvc,
		webhookSvc:       webhookSvc,
		storageSvc:       storageSvc,
		storageScheduler: storageScheduler,
	}
}

// ============ 系统信息 API ============

// GetSystemInfo 获取管理员可配置的系统信息
//
// @Summary      获取系统信息
// @Description  获取管理员可配置的系统信息（设备型号、部署位置、描述）
// @Tags         系统管理
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/info [get]
// @Security     BearerAuth
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	info, err := h.systemInfoSvc.GetSystemInfo()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, info)
}

// UpdateSystemInfo 更新系统信息
//
// @Summary      更新系统信息
// @Description  更新管理员可配置的系统信息
// @Tags         系统管理
// @Accept       json
// @Produce      json
// @Param        body  body  service.SystemInfo  true  "系统信息"
// @Success      200   {object}  map[string]interface{}
// @Router       /system/info [put]
// @Security     BearerAuth
func (h *SystemHandler) UpdateSystemInfo(c *gin.Context) {
	var req service.SystemInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}
	if err := h.systemInfoSvc.UpdateSystemInfo(&req); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, nil)
}

// ============ 运行状态 API ============

// GetRealtimeStatus 获取实时指标
//
// @Summary      获取实时指标
// @Description  获取 CPU、内存、运行时长、任务等实时指标
// @Tags         系统管理-运行状态
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/status/realtime [get]
// @Security     BearerAuth
func (h *SystemHandler) GetRealtimeStatus(c *gin.Context) {
	metrics, err := h.systemSvc.GetRealtimeStatus()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, metrics)
}

// GetResourceStatus 获取资源指标
//
// @Summary      获取资源指标
// @Description  获取 NPU、磁盘、网络等资源指标
// @Tags         系统管理-运行状态
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/status/resources [get]
// @Security     BearerAuth
func (h *SystemHandler) GetResourceStatus(c *gin.Context) {
	metrics, err := h.systemSvc.GetResourceStatus()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, metrics)
}

// GetServiceStatus 获取服务状态
//
// @Summary      获取服务状态
// @Description  获取 Go管理端、C++引擎、ZLMediaKit、Redis、PostgreSQL 的运行状态
// @Tags         系统管理-运行状态
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/status/services [get]
// @Security     BearerAuth
func (h *SystemHandler) GetServiceStatus(c *gin.Context) {
	status, err := h.systemSvc.GetServiceStatus()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, status)
}

// GetEngineStatus 获取引擎全局指标
//
// @Summary      获取引擎全局指标
// @Description  获取 C++ 推理引擎的全局运行指标（DMA/NPU 内存、Worker 数等）
// @Tags         系统管理-运行状态
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/status/engine [get]
// @Security     BearerAuth
func (h *SystemHandler) GetEngineStatus(c *gin.Context) {
	status := h.systemSvc.GetEngineStatus()
	response.OK(c, status)
}

// GetStreamsStatus 获取各流推理指标
//
// @Summary      获取各流推理指标
// @Description  获取 C++ 推理引擎各视频流的推理延迟、队列深度等指标
// @Tags         系统管理-运行状态
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/status/streams [get]
// @Security     BearerAuth
func (h *SystemHandler) GetStreamsStatus(c *gin.Context) {
	streams := h.systemSvc.GetStreamsStatus()
	response.OK(c, streams)
}

// GetStatusHistory 获取历史数据
//
// @Summary      获取历史指标
// @Description  获取指定指标在指定时间范围内的历史数据点（支持 engine.* 前缀指标）
// @Tags         系统管理-运行状态
// @Produce      json
// @Param        metric    query  string  true   "指标名称 (cpu/memory/npu)"
// @Param        duration  query  string  false  "时间范围 (如 5m, 1h)"  default(5m)
// @Success      200  {object}  map[string]interface{}
// @Router       /system/status/history [get]
// @Security     BearerAuth
func (h *SystemHandler) GetStatusHistory(c *gin.Context) {
	metric := c.Query("metric")
	if metric == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "missing required parameter: metric"))
		return
	}

	durationStr := c.DefaultQuery("duration", "5m")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "invalid duration format"))
		return
	}

	points, err := h.systemSvc.GetStatusHistory(metric, duration)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, gin.H{
		"metric": metric,
		"points": points,
	})
}

// ============ 网络配置 API ============

// GetNetworkInterfaces 获取所有网卡状态
//
// @Summary      获取网卡列表
// @Description  获取所有网络接口的状态和配置信息
// @Tags         系统管理-网络配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/network [get]
// @Security     BearerAuth
func (h *SystemHandler) GetNetworkInterfaces(c *gin.Context) {
	interfaces, err := h.networkSvc.GetInterfaces()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, interfaces)
}

// ApplyNetworkConfig 应用网络配置
//
// @Summary      应用网络配置
// @Description  应用网络接口配置，返回事务 ID 用于确认或回滚（超时 120 秒未确认自动回滚）
// @Tags         系统管理-网络配置
// @Accept       json
// @Produce      json
// @Param        body  body  service.NetworkConfigRequest  true  "网络配置"
// @Success      200   {object}  map[string]interface{}
// @Router       /system/network/apply [post]
// @Security     BearerAuth
func (h *SystemHandler) ApplyNetworkConfig(c *gin.Context) {
	var req service.NetworkConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}

	transaction, err := h.networkSvc.ApplyConfig(&req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, gin.H{
		"transaction_id":   transaction.TransactionID,
		"rollback_timeout": 120,
		"confirm_url":      "/api/v1/system/network/confirm?transaction_id=" + transaction.TransactionID,
	})
}

// ConfirmNetworkConfig 确认网络配置生效
//
// @Summary      确认网络配置
// @Description  确认网络配置生效，取消自动回滚
// @Tags         系统管理-网络配置
// @Produce      json
// @Param        transaction_id  query  string  true  "事务 ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /system/network/confirm [post]
// @Security     BearerAuth
func (h *SystemHandler) ConfirmNetworkConfig(c *gin.Context) {
	transactionID := c.Query("transaction_id")
	if transactionID == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "missing required parameter: transaction_id"))
		return
	}

	if err := h.networkSvc.ConfirmConfig(transactionID); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// RollbackNetworkConfig 回滚网络配置
//
// @Summary      回滚网络配置
// @Description  手动回滚到之前的网络配置
// @Tags         系统管理-网络配置
// @Produce      json
// @Param        transaction_id  query  string  true  "事务 ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /system/network/rollback [post]
// @Security     BearerAuth
func (h *SystemHandler) RollbackNetworkConfig(c *gin.Context) {
	transactionID := c.Query("transaction_id")
	if transactionID == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "missing required parameter: transaction_id"))
		return
	}

	if err := h.networkSvc.RollbackConfig(transactionID); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// ============ 时间配置 API ============

// GetTimeConfig 获取时间配置
//
// @Summary      获取时间配置
// @Description  获取当前时间、时区、NTP 配置等信息
// @Tags         系统管理-时间配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/time [get]
// @Security     BearerAuth
func (h *SystemHandler) GetTimeConfig(c *gin.Context) {
	config, err := h.timeConfigSvc.GetTimeConfig()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, config)
}

// SetManualTime 手动设置时间
//
// @Summary      手动设置时间
// @Description  手动设置系统时间（偏差不得超过 ±24 小时）
// @Tags         系统管理-时间配置
// @Accept       json
// @Produce      json
// @Param        body  body  object{time=string}  true  "时间 (RFC3339 格式)"
// @Success      200   {object}  map[string]interface{}
// @Router       /system/time/manual [post]
// @Security     BearerAuth
func (h *SystemHandler) SetManualTime(c *gin.Context) {
	var req struct {
		Time time.Time `json:"time" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}

	if err := h.timeConfigSvc.SetManualTime(req.Time); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// GetTimezone 获取时区
//
// @Summary      获取时区
// @Description  获取当前系统时区
// @Tags         系统管理-时间配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/time/timezone [get]
// @Security     BearerAuth
func (h *SystemHandler) GetTimezone(c *gin.Context) {
	timezone, err := h.timeConfigSvc.GetTimezone()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, gin.H{"timezone": timezone})
}

// SetTimezone 设置时区
//
// @Summary      设置时区
// @Description  设置系统时区（如 Asia/Shanghai）
// @Tags         系统管理-时间配置
// @Accept       json
// @Produce      json
// @Param        body  body  object{timezone=string}  true  "时区名称"
// @Success      200   {object}  map[string]interface{}
// @Router       /system/time/timezone [put]
// @Security     BearerAuth
func (h *SystemHandler) SetTimezone(c *gin.Context) {
	var req struct {
		Timezone string `json:"timezone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}

	if err := h.timeConfigSvc.SetTimezone(req.Timezone); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// GetNTPConfig 获取 NTP 配置
//
// @Summary      获取 NTP 配置
// @Description  获取 NTP 服务器配置和同步状态
// @Tags         系统管理-时间配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/time/ntp [get]
// @Security     BearerAuth
func (h *SystemHandler) GetNTPConfig(c *gin.Context) {
	config, err := h.timeConfigSvc.GetNTPConfig()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, config)
}

// AddNTPServer 添加 NTP 服务器
//
// @Summary      添加 NTP 服务器
// @Description  添加 NTP 服务器到配置列表
// @Tags         系统管理-时间配置
// @Accept       json
// @Produce      json
// @Param        body  body  object{host=string}  true  "NTP 服务器地址"
// @Success      200   {object}  map[string]interface{}
// @Router       /system/time/ntp/servers [post]
// @Security     BearerAuth
func (h *SystemHandler) AddNTPServer(c *gin.Context) {
	var req struct {
		Host string `json:"host" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}

	if err := h.timeConfigSvc.AddNTPServer(req.Host); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// RemoveNTPServer 删除 NTP 服务器
//
// @Summary      删除 NTP 服务器
// @Description  从配置列表中移除指定 NTP 服务器
// @Tags         系统管理-时间配置
// @Produce      json
// @Param        host  path  string  true  "NTP 服务器地址"
// @Success      200  {object}  map[string]interface{}
// @Router       /system/time/ntp/servers/{host} [delete]
// @Security     BearerAuth
func (h *SystemHandler) RemoveNTPServer(c *gin.Context) {
	host := c.Param("host")
	if host == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "missing required parameter: host"))
		return
	}

	if err := h.timeConfigSvc.RemoveNTPServer(host); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// TestNTPServers 测试 NTP 服务器连通性
//
// @Summary      测试 NTP 服务器
// @Description  并发测试所有已配置 NTP 服务器的可达性
// @Tags         系统管理-时间配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/time/ntp/test [post]
// @Security     BearerAuth
func (h *SystemHandler) TestNTPServers(c *gin.Context) {
	results, err := h.timeConfigSvc.TestNTPServers()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, results)
}

// SyncNTP 立即同步 NTP
//
// @Summary      立即同步 NTP
// @Description  立即从 NTP 服务器同步时间
// @Tags         系统管理-时间配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/time/ntp/sync [post]
// @Security     BearerAuth
func (h *SystemHandler) SyncNTP(c *gin.Context) {
	if err := h.timeConfigSvc.SyncNTP(); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, nil)
}

// GetRecommendedNTPServers 获取推荐 NTP 服务器列表
//
// @Summary      获取推荐 NTP 服务器
// @Description  获取推荐的 NTP 服务器列表
// @Tags         系统管理-时间配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/time/ntp/recommended [get]
// @Security     BearerAuth
func (h *SystemHandler) GetRecommendedNTPServers(c *gin.Context) {
	servers := h.timeConfigSvc.GetRecommendedNTPServers()
	response.OK(c, servers)
}

// ============ 告警上报 API ============

// ListWebhooks 获取 Webhook 列表
//
// @Summary      Webhook 列表
// @Description  获取所有 Webhook 告警推送配置
// @Tags         系统管理-告警上报
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/webhook [get]
// @Security     BearerAuth
func (h *SystemHandler) ListWebhooks(c *gin.Context) {
	webhooks, err := h.webhookSvc.List()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, webhooks)
}

// CreateWebhook 创建 Webhook
//
// @Summary      创建 Webhook
// @Description  创建新的 Webhook 告警推送配置
// @Tags         系统管理-告警上报
// @Accept       json
// @Produce      json
// @Param        body  body  service.WebhookCreateRequest  true  "Webhook 配置"
// @Success      200   {object}  map[string]interface{}
// @Router       /system/webhook [post]
// @Security     BearerAuth
func (h *SystemHandler) CreateWebhook(c *gin.Context) {
	var req service.WebhookCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}

	webhook, err := h.webhookSvc.Create(&req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, webhook)
}

// GetWebhook 获取 Webhook 详情
//
// @Summary      Webhook 详情
// @Description  根据 ID 获取 Webhook 配置详情
// @Tags         系统管理-告警上报
// @Produce      json
// @Param        id  path  string  true  "Webhook ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /system/webhook/{id} [get]
// @Security     BearerAuth
func (h *SystemHandler) GetWebhook(c *gin.Context) {
	id := c.Param("id")
	webhook, err := h.webhookSvc.GetByID(id)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, webhook)
}

// UpdateWebhook 更新 Webhook
//
// @Summary      更新 Webhook
// @Description  更新指定 Webhook 的配置
// @Tags         系统管理-告警上报
// @Accept       json
// @Produce      json
// @Param        id    path  string                    true  "Webhook ID"
// @Param        body  body  service.WebhookCreateRequest  true  "Webhook 配置"
// @Success      200   {object}  map[string]interface{}
// @Router       /system/webhook/{id} [put]
// @Security     BearerAuth
func (h *SystemHandler) UpdateWebhook(c *gin.Context) {
	id := c.Param("id")
	var req service.WebhookCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}

	webhook, err := h.webhookSvc.Update(id, &req)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, webhook)
}

// DeleteWebhook 删除 Webhook
//
// @Summary      删除 Webhook
// @Description  删除指定 Webhook 配置
// @Tags         系统管理-告警上报
// @Produce      json
// @Param        id  path  string  true  "Webhook ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /system/webhook/{id} [delete]
// @Security     BearerAuth
func (h *SystemHandler) DeleteWebhook(c *gin.Context) {
	id := c.Param("id")
	if err := h.webhookSvc.Delete(id); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, nil)
}

// TestWebhook 测试 Webhook
//
// @Summary      测试 Webhook
// @Description  发送测试请求到指定 Webhook
// @Tags         系统管理-告警上报
// @Produce      json
// @Param        id  path  string  true  "Webhook ID"
// @Success      200  {object}  map[string]interface{}
// @Router       /system/webhook/{id}/test [post]
// @Security     BearerAuth
func (h *SystemHandler) TestWebhook(c *gin.Context) {
	id := c.Param("id")
	result, err := h.webhookSvc.Test(id)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, result)
}

// ListWebhookLogs 获取推送日志
//
// @Summary      Webhook 推送日志
// @Description  分页查询 Webhook 推送日志
// @Tags         系统管理-告警上报
// @Produce      json
// @Param        webhook_id  query  string  false  "Webhook ID 筛选"
// @Param        page        query  int     false  "页码"       default(1)
// @Param        page_size   query  int     false  "每页数量"   default(20)
// @Success      200  {object}  dto.Response{data=dto.PageData}
// @Router       /system/webhook/logs [get]
// @Security     BearerAuth
func (h *SystemHandler) ListWebhookLogs(c *gin.Context) {
	webhookID := c.Query("webhook_id")
	page, pageSize := parsePageParams(c)

	logs, total, err := h.webhookSvc.ListLogs(webhookID, page, pageSize)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, logs, total, page, pageSize)
}

// ============ 存储配置 API ============

// GetStorageConfig 获取存储配置
//
// @Summary      获取存储配置
// @Description  获取存储清理策略配置（CRON / 阈值模式）
// @Tags         系统管理-存储配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/storage/config [get]
// @Security     BearerAuth
func (h *SystemHandler) GetStorageConfig(c *gin.Context) {
	config, err := h.storageSvc.GetConfig()
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, config)
}

// UpdateStorageConfig 更新存储配置
//
// @Summary      更新存储配置
// @Description  更新存储清理策略配置并自动重建调度
// @Tags         系统管理-存储配置
// @Accept       json
// @Produce      json
// @Param        body  body  service.StorageConfig  true  "存储配置"
// @Success      200   {object}  map[string]interface{}
// @Router       /system/storage/config [put]
// @Security     BearerAuth
func (h *SystemHandler) UpdateStorageConfig(c *gin.Context) {
	var req service.StorageConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, err)
		return
	}

	if err := h.storageSvc.UpdateConfig(&req); err != nil {
		response.Err(c, err)
		return
	}

	// 通知 StorageScheduler 重建调度
	if h.storageScheduler != nil {
		if err := h.storageScheduler.UpdateConfig(&req); err != nil {
			// 调度更新失败不影响配置保存，仅记录日志
			zap.L().Warn("failed to update storage scheduler", zap.Error(err))
		}
	}

	response.OK(c, nil)
}

// ListCleanupLogs 获取清理日志
//
// @Summary      清理日志列表
// @Description  分页查询存储清理日志
// @Tags         系统管理-存储配置
// @Produce      json
// @Param        page       query  int  false  "页码"       default(1)
// @Param        page_size  query  int  false  "每页数量"   default(20)
// @Success      200  {object}  dto.Response{data=dto.PageData}
// @Router       /system/storage/cleanup-logs [get]
// @Security     BearerAuth
func (h *SystemHandler) ListCleanupLogs(c *gin.Context) {
	page, pageSize := parsePageParams(c)

	logs, total, err := h.storageSvc.ListCleanupLogs(page, pageSize)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, logs, total, page, pageSize)
}

// RunCleanup 手动触发清理
//
// @Summary      手动触发清理
// @Description  手动触发一次存储清理操作
// @Tags         系统管理-存储配置
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /system/storage/cleanup/run [post]
// @Security     BearerAuth
func (h *SystemHandler) RunCleanup(c *gin.Context) {
	if err := h.storageSvc.RunCleanup(); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// parsePageParams 从请求中解析分页参数
func parsePageParams(c *gin.Context) (page, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return
}
