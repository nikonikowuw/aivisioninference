package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
)

type gb28181Service interface {
	ListGB28181Devices(ctx context.Context, req dto.GB28181DeviceListRequest) ([]model.GB28181Device, int64, error)
	GetGB28181DeviceByID(ctx context.Context, id string) (*model.GB28181Device, error)
	UpdateGB28181Device(ctx context.Context, id string, req dto.GB28181DeviceUpdateRequest) (*model.GB28181Device, error)
	DeleteGB28181Device(ctx context.Context, id string) error
	QueryCatalog(ctx context.Context, deviceCode string) error
	GetGB28181DeviceChannels(ctx context.Context, id string) ([]model.Device, error)
}

type GB28181Handler struct {
	sipSvc gb28181Service
	cache  cache.Cache
	hub    *ws.Hub
}

func NewGB28181Handler(sipSvc gb28181Service, cache cache.Cache, hub *ws.Hub) *GB28181Handler {
	return &GB28181Handler{
		sipSvc: sipSvc,
		cache:  cache,
		hub:    hub,
	}
}

// List GB28181 设备列表
// @Summary      GB28181 设备列表
// @Tags         GB28181管理
// @Produce      json
// @Param        page        query   int     false  "页码"
// @Param        page_size   query   int     false  "每页数量"
// @Param        keyword     query   string  false  "关键词搜索"
// @Param        status      query   string  false  "状态筛选"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.GB28181DeviceResponse}}
// @Router       /gb28181/devices [get]
// @Security     BearerAuth
func (h *GB28181Handler) List(c *gin.Context) {
	var req dto.GB28181DeviceListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	items, total, err := h.sipSvc.ListGB28181Devices(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	var list []dto.GB28181DeviceResponse
	for _, item := range items {
		list = append(list, dto.GB28181DeviceResponse{
			ID:                item.ID,
			DeviceCode:        item.DeviceCode,
			RegisterAddress:   item.RegisterAddress,
			RegisterPort:      item.RegisterPort,
			SipID:             item.SipID,
			SipDomain:         item.SipDomain,
			LastRegisterAt:    item.LastRegisterAt,
			LastHeartbeatAt:   item.LastHeartbeatAt,
			LastCatalogAt:     item.LastCatalogAt,
			HeartbeatInterval: item.HeartbeatInterval,
			Status:            item.Status,
			ChannelCount:      item.ChannelCount,
			Manufacturer:      item.Manufacturer,
			Model:             item.Model,
			Firmware:          item.Firmware,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		})
	}

	response.Page(c, list, total, req.GetPage(), req.GetPageSize())
}

// GetByID GB28181 设备详情
// @Summary      GB28181 设备详情
// @Tags         GB28181管理
// @Produce      json
// @Param        id  path  string  true  "设备 ID"
// @Success      200  {object}  dto.Response{data=dto.GB28181DeviceResponse}
// @Router       /gb28181/devices/{id} [get]
// @Security     BearerAuth
func (h *GB28181Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	item, err := h.sipSvc.GetGB28181DeviceByID(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, dto.GB28181DeviceResponse{
		ID:                item.ID,
		DeviceCode:        item.DeviceCode,
		RegisterAddress:   item.RegisterAddress,
		RegisterPort:      item.RegisterPort,
		SipID:             item.SipID,
		SipDomain:         item.SipDomain,
		LastRegisterAt:    item.LastRegisterAt,
		LastHeartbeatAt:   item.LastHeartbeatAt,
		LastCatalogAt:     item.LastCatalogAt,
		HeartbeatInterval: item.HeartbeatInterval,
		Status:            item.Status,
		ChannelCount:      item.ChannelCount,
		Manufacturer:      item.Manufacturer,
		Model:             item.Model,
		Firmware:          item.Firmware,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	})
}

// Update 更新 GB28181 设备
// @Summary      更新 GB28181 设备
// @Tags         GB28181管理
// @Accept       json
// @Produce      json
// @Param        id    path   string                   true  "设备 ID"
// @Param        body  body   dto.GB28181DeviceUpdateRequest  true  "更新信息"
// @Success      200   {object}  dto.Response{data=dto.GB28181DeviceResponse}
// @Router       /gb28181/devices/{id} [put]
// @Security     BearerAuth
func (h *GB28181Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.GB28181DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	item, err := h.sipSvc.UpdateGB28181Device(c.Request.Context(), id, req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, dto.GB28181DeviceResponse{
		ID:                item.ID,
		DeviceCode:        item.DeviceCode,
		RegisterAddress:   item.RegisterAddress,
		RegisterPort:      item.RegisterPort,
		SipID:             item.SipID,
		SipDomain:         item.SipDomain,
		LastRegisterAt:    item.LastRegisterAt,
		LastHeartbeatAt:   item.LastHeartbeatAt,
		LastCatalogAt:     item.LastCatalogAt,
		HeartbeatInterval: item.HeartbeatInterval,
		Status:            item.Status,
		ChannelCount:      item.ChannelCount,
		Manufacturer:      item.Manufacturer,
		Model:             item.Model,
		Firmware:          item.Firmware,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	})
}

// Delete 删除 GB28181 设备
// @Summary      删除 GB28181 设备
// @Tags         GB28181管理
// @Produce      json
// @Param        id  path  string  true  "设备 ID"
// @Success      200  {object}  dto.Response
// @Router       /gb28181/devices/{id} [delete]
// @Security     BearerAuth
func (h *GB28181Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.sipSvc.DeleteGB28181Device(c.Request.Context(), id); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// TriggerCatalog 触发目录查询
// @Summary      触发目录查询
// @Tags         GB28181管理
// @Produce      json
// @Param        id  path  string  true  "设备 ID"
// @Success      200  {object}  dto.Response{data=dto.CatalogTaskResponse}
// @Router       /gb28181/devices/{id}/catalog [post]
// @Security     BearerAuth
func (h *GB28181Handler) TriggerCatalog(c *gin.Context) {
	id := c.Param("id")
	item, err := h.sipSvc.GetGB28181DeviceByID(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}

	taskID := uuid.New().String()
	cacheKey := "catalog_task:" + taskID
	deviceTaskKey := "catalog_task:device:" + item.DeviceCode

	status := dto.CatalogTaskStatusResponse{
		TaskID: taskID,
		Status: "pending",
	}
	statusData, _ := json.Marshal(status)
	_ = h.cache.Set(context.Background(), cacheKey, statusData, 2*time.Minute)
	_ = h.cache.Set(context.Background(), deviceTaskKey, []byte(taskID), 2*time.Minute)

	if err := h.sipSvc.QueryCatalog(c.Request.Context(), item.DeviceCode); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, dto.CatalogTaskResponse{TaskID: taskID})
}

// GetChannels 通道列表接口
// @Summary      通道列表接口
// @Tags         GB28181管理
// @Produce      json
// @Param        id  path  string  true  "设备 ID"
// @Success      200  {object}  dto.Response{data=[]dto.DeviceResponse}
// @Router       /gb28181/devices/{id}/channels [get]
// @Security     BearerAuth
func (h *GB28181Handler) GetChannels(c *gin.Context) {
	id := c.Param("id")
	devices, err := h.sipSvc.GetGB28181DeviceChannels(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}

	var list []dto.DeviceResponse
	for _, item := range devices {
		list = append(list, dto.DeviceResponse{
			ID:               item.ID,
			DeviceName:       item.DeviceName,
			AccessType:       item.AccessType,
			GB28181DeviceID:  item.GB28181DeviceID,
			GB28181ChannelID: item.GB28181ChannelID,
			Status:           item.Status,
			Enabled:          item.Enabled,
			Manufacturer:     item.Manufacturer,
			Model:            item.Model,
			FirmwareVersion:  item.FirmwareVersion,
			CreatedAt:        item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        item.UpdatedAt.Format(time.RFC3339),
		})
	}
	response.OK(c, list)
}

// GetCatalogTaskStatus 状态查询接口
// @Summary      目录查询任务状态
// @Tags         GB28181管理
// @Produce      json
// @Param        task_id  path  string  true  "任务 ID"
// @Success      200  {object}  dto.Response{data=dto.CatalogTaskStatusResponse}
// @Router       /gb28181/catalog-tasks/{task_id} [get]
// @Security     BearerAuth
func (h *GB28181Handler) GetCatalogTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	data, err := h.cache.Get(context.Background(), "catalog_task:"+taskID)
	if err != nil {
		// If not found, it might have expired or never existed
		attachError(c, err)
		return
	}

	var status dto.CatalogTaskStatusResponse
	_ = json.Unmarshal(data, &status)

	response.OK(c, status)
}
