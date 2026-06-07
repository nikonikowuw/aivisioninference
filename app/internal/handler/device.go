// Package handler 提供 HTTP 请求处理层（Controller），负责参数绑定、校验和响应返回。
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// DeviceHandler 处理设备管理相关的 HTTP 请求
type DeviceHandler struct {
	svc *service.DeviceService
}

// NewDeviceHandler 创建一个新的 DeviceHandler 实例
func NewDeviceHandler(svc *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{svc: svc}
}

// List 返回分页的设备列表
//
// @Summary      设备列表
// @Description  分页查询设备列表，支持按关键词、状态、接入类型、分组筛选
// @Tags         设备管理
// @Produce      json
// @Param        page        query   int     false  "页码"       default(1)
// @Param        page_size   query   int     false  "每页数量"   default(20)
// @Param        keyword     query   string  false  "关键词搜索（设备名称/制造商/型号）"
// @Param        status      query   string  false  "状态筛选 (unknown/online/offline/error/disabled)"
// @Param        access_type query   string  false  "接入类型筛选 (rtsp/gb28181/nvr_channel/other)"
// @Param        group_id    query   string  false  "设备分组 ID"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.DeviceListResponse}}
// @Router       /devices [get]
// @Security     BearerAuth
func (h *DeviceHandler) List(c *gin.Context) {
	var req dto.DeviceListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// GetByID 根据设备 ID 获取设备详情
//
// @Summary      设备详情
// @Description  根据 ID 查询设备详细信息
// @Tags         设备管理
// @Produce      json
// @Param        id  path  string  true  "设备 ID"
// @Success      200  {object}  dto.Response{data=dto.DeviceResponse}
// @Router       /devices/{id} [get]
// @Security     BearerAuth
func (h *DeviceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	device, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, device)
}

// Create 创建新的设备
//
// @Summary      创建设备
// @Description  创建新的流设备，支持 RTSP、GB28181 等接入类型
// @Tags         设备管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.DeviceCreateRequest  true  "设备信息"
// @Success      200   {object}  dto.Response{data=dto.DeviceResponse}
// @Router       /devices [post]
// @Security     BearerAuth
func (h *DeviceHandler) Create(c *gin.Context) {
	var req dto.DeviceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	device, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, device)
}

// Update 更新设备信息
//
// @Summary      更新设备
// @Description  更新设备信息，密码留空表示不修改
// @Tags         设备管理
// @Accept       json
// @Produce      json
// @Param        id    path   string                   true  "设备 ID"
// @Param        body  body   dto.DeviceUpdateRequest  true  "设备信息"
// @Success      200   {object}  dto.Response{data=dto.DeviceResponse}
// @Router       /devices/{id} [put]
// @Security     BearerAuth
func (h *DeviceHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	device, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, device)
}

// Delete 删除设备（软删除）
//
// @Summary      删除设备
// @Description  软删除设备，已绑定任务的设备无法删除
// @Tags         设备管理
// @Produce      json
// @Param        id  path  string  true  "设备 ID"
// @Success      200  {object}  dto.Response
// @Router       /devices/{id} [delete]
// @Security     BearerAuth
func (h *DeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// BatchDelete 批量删除设备
//
// @Summary      批量删除设备
// @Description  批量软删除设备
// @Tags         设备管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.BatchIDsRequest  true  "设备 ID 列表"
// @Success      200   {object}  dto.Response{data=dto.BatchResult}
// @Router       /devices/batch-delete [post]
// @Security     BearerAuth
func (h *DeviceHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	response.OK(c, h.svc.BatchDelete(c.Request.Context(), req.IDs))
}

// TestConnection 测试设备连接
//
// @Summary      测试设备连接
// @Description  测试设备的 RTSP/GB28181 连接状态
// @Tags         设备管理
// @Produce      json
// @Param        id  path  string  true  "设备 ID"
// @Success      200  {object}  dto.Response{data=dto.DeviceTestResultResponse}
// @Router       /devices/{id}/test [post]
// @Security     BearerAuth
func (h *DeviceHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")
	result, err := h.svc.TestConnection(c.Request.Context(), id, currentLang(c))
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, result)
}

// ExportCSV 导出设备列表为 CSV
//
// @Summary      导出设备 CSV
// @Description  按当前筛选条件导出设备列表为 CSV 文件
// @Tags         设备管理
// @Produce      text/csv
// @Success      200  {file}  file
// @Router       /devices/export [get]
// @Security     BearerAuth
func (h *DeviceHandler) ExportCSV(c *gin.Context) {
	var req dto.DeviceListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	data, err := h.svc.ExportCSV(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	writeCSV(c, "devices.csv", data)
}

// ImportCSV 从 CSV 文件批量导入设备
//
// @Summary      导入设备 CSV
// @Description  通过 CSV 文件批量导入设备
// @Tags         设备管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "CSV 文件"
// @Success      200   {object}  dto.Response{data=dto.BatchResult}
// @Router       /devices/import [post]
// @Security     BearerAuth
func (h *DeviceHandler) ImportCSV(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}
	if fileHeader.Size > maxCSVImportSize {
		attachError(c, apperrors.New(apperrors.ErrFileTooLarge, ""))
		return
	}
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".csv") {
		attachError(c, apperrors.New(apperrors.ErrFileInvalidType, ""))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}
	defer file.Close()
	result, err := h.svc.ImportCSV(c.Request.Context(), file, currentLang(c))
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, result)
}

// DeviceGroupHandler 处理设备分组相关的 HTTP 请求
type DeviceGroupHandler struct {
	svc *service.DeviceGroupService
}

// NewDeviceGroupHandler 创建一个新的 DeviceGroupHandler 实例
func NewDeviceGroupHandler(svc *service.DeviceGroupService) *DeviceGroupHandler {
	return &DeviceGroupHandler{svc: svc}
}

// List 返回分页的设备分组列表
//
// @Summary      设备分组列表
// @Description  分页查询设备分组列表，包含各分组的设备数量
// @Tags         设备分组管理
// @Produce      json
// @Param        page      query   int     false  "页码"       default(1)
// @Param        page_size query   int     false  "每页数量"   default(20)
// @Param        keyword   query   string  false  "关键词搜索（分组名称）"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.DeviceGroupResponse}}
// @Router       /device-groups [get]
// @Security     BearerAuth
func (h *DeviceGroupHandler) List(c *gin.Context) {
	var req dto.DeviceGroupListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// GetByID 根据分组 ID 获取设备分组详情
//
// @Summary      设备分组详情
// @Description  根据 ID 查询设备分组详细信息
// @Tags         设备分组管理
// @Produce      json
// @Param        id  path  string  true  "分组 ID"
// @Success      200  {object}  dto.Response{data=dto.DeviceGroupResponse}
// @Router       /device-groups/{id} [get]
// @Security     BearerAuth
func (h *DeviceGroupHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	group, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, group)
}

// Create 创建新的设备分组
//
// @Summary      创建设备分组
// @Description  创建新的设备分组
// @Tags         设备分组管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.DeviceGroupCreateRequest  true  "分组信息"
// @Success      200   {object}  dto.Response{data=dto.DeviceGroupResponse}
// @Router       /device-groups [post]
// @Security     BearerAuth
func (h *DeviceGroupHandler) Create(c *gin.Context) {
	var req dto.DeviceGroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	group, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, group)
}

// Update 更新设备分组信息
//
// @Summary      更新设备分组
// @Description  更新设备分组信息
// @Tags         设备分组管理
// @Accept       json
// @Produce      json
// @Param        id    path   string                      true  "分组 ID"
// @Param        body  body   dto.DeviceGroupUpdateRequest true  "分组信息"
// @Success      200   {object}  dto.Response{data=dto.DeviceGroupResponse}
// @Router       /device-groups/{id} [put]
// @Security     BearerAuth
func (h *DeviceGroupHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.DeviceGroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	group, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, group)
}

// Delete 删除设备分组
//
// @Summary      删除设备分组
// @Description  删除设备分组，分组下有设备时无法删除
// @Tags         设备分组管理
// @Produce      json
// @Param        id  path  string  true  "分组 ID"
// @Success      200  {object}  dto.Response
// @Router       /device-groups/{id} [delete]
// @Security     BearerAuth
func (h *DeviceGroupHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}
