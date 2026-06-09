package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

type DeviceStagingHandler struct {
	svc          *service.DeviceStagingService
	discoverySvc *service.DeviceDiscoveryService
}

func NewDeviceStagingHandler(svc *service.DeviceStagingService, discoverySvc *service.DeviceDiscoveryService) *DeviceStagingHandler {
	return &DeviceStagingHandler{
		svc:          svc,
		discoverySvc: discoverySvc,
	}
}

func (h *DeviceStagingHandler) List(c *gin.Context) {
	source := c.Query("source")
	status := c.Query("status")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 分页参数边界校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	items, total, err := h.svc.List(c.Request.Context(), source, status, keyword, page, pageSize)
	if err != nil {
		response.Err(c, apperrors.New(apperrors.ErrInternal, "获取暂存设备列表失败"))
		return
	}
	response.Page(c, items, total, page, pageSize)
}

func (h *DeviceStagingHandler) ScanONVIF(c *gin.Context) {
	netInterface := c.Query("interface")
	if netInterface == "" {
		netInterface = "eth0"
	}

	if err := h.discoverySvc.DiscoverONVIF(c.Request.Context(), netInterface); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrInternal, "ONVIF 扫描失败"))
		return
	}
	response.OK(c, nil)
}

func (h *DeviceStagingHandler) BatchImport(c *gin.Context) {
	var req dto.BatchImportDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}
	if err := h.svc.BatchImport(c.Request.Context(), req.IDs, req.Username, req.Password, req.EnableInfer); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrInternal, "批量导入部分失败"))
		return
	}
	response.OK(c, nil)
}

func (h *DeviceStagingHandler) BatchIgnore(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}
	if err := h.svc.BatchIgnore(c.Request.Context(), req.IDs); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrInternal, "批量忽略失败"))
		return
	}
	response.OK(c, nil)
}

func (h *DeviceStagingHandler) Import(c *gin.Context) {
	id := c.Param("id")

	var req dto.ImportDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.svc.ImportSingle(c.Request.Context(), id, req.Username, req.Password, req.DeviceName, req.EnableInfer); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrInternal, "导入设备失败"))
		return
	}
	response.OK(c, nil)
}

func (h *DeviceStagingHandler) Ignore(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.BatchIgnore(c.Request.Context(), []string{id}); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrInternal, "忽略设备失败"))
		return
	}
	response.OK(c, nil)
}
