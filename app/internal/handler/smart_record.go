package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// SmartRecordHandler 处理智能记录相关请求。
type SmartRecordHandler struct {
	svc *service.SmartRecordService
}

// NewSmartRecordHandler 创建智能记录 Handler。
func NewSmartRecordHandler(svc *service.SmartRecordService) *SmartRecordHandler {
	return &SmartRecordHandler{svc: svc}
}

// ListCategoryCodes 返回所有启用的类别编码供前端下拉使用。
//
// @Summary      类别编码下拉列表
// @Tags         智能记录
// @Produce      json
// @Success      200  {object}  dto.Response{data=[]dto.CategoryCodeOption}
// @Router       /smart-records/category-codes [get]
// @Security     BearerAuth
func (h *SmartRecordHandler) ListCategoryCodes(c *gin.Context) {
	items, err := h.svc.ListCategoryCodes(c.Request.Context())
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, items)
}

// List 智能记录列表。
//
// @Summary      智能记录列表
// @Description  分页查询识别记录、告警记录和抓拍记录
// @Tags         智能记录
// @Produce      json
// @Param        page          query   int     false  "页码"      default(1)
// @Param        page_size     query   int     false  "每页数量"  default(20)
// @Param        type          query   string  false  "记录类型 recognition|alarm|capture"
// @Param        keyword       query   string  false  "关键词"
// @Param        device_id     query   string  false  "设备 ID"
// @Param        device_name   query   string  false  "设备名称"
// @Param        alarm_type    query   string  false  "告警类型"
// @Param        alarm_level   query   string  false  "告警级别"
// @Param        person_name   query   string  false  "人员姓名"
// @Param        start_time    query   string  false  "开始时间"
// @Param        end_time      query   string  false  "结束时间"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.SmartRecord}}
// @Router       /smart-records [get]
// @Security     BearerAuth
func (h *SmartRecordHandler) List(c *gin.Context) {
	var req dto.SmartRecordListRequest
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

// ExportCSV 按当前筛选条件导出智能记录 CSV。
//
// @Summary      导出智能记录 CSV
// @Description  根据当前列表筛选条件导出智能记录为 CSV 文件
// @Tags         智能记录
// @Produce      text/csv
// @Param        page          query   int     false  "页码"      default(1)
// @Param        page_size     query   int     false  "每页数量"  default(20)
// @Param        type          query   string  false  "记录类型 recognition|alarm|capture"
// @Param        keyword       query   string  false  "关键词"
// @Param        device_id     query   string  false  "设备 ID"
// @Param        device_name   query   string  false  "设备名称"
// @Param        alarm_type    query   string  false  "告警类型"
// @Param        alarm_level   query   string  false  "告警级别"
// @Param        person_name   query   string  false  "人员姓名"
// @Param        start_time    query   string  false  "开始时间"
// @Param        end_time      query   string  false  "结束时间"
// @Success      200  {file}    file    "CSV 文件"
// @Router       /smart-records/export [get]
// @Security     BearerAuth
func (h *SmartRecordHandler) ExportCSV(c *gin.Context) {
	var req dto.SmartRecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	data, err := h.svc.ExportCSV(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	writeCSV(c, "smart-records.csv", data)
}

// BatchDelete 批量删除智能记录。
//
// @Summary      批量删除智能记录
// @Tags         智能记录
// @Accept       json
// @Produce      json
// @Param        body  body  dto.BatchIDsRequest  true  "记录 ID 列表"
// @Success      200   {object}  dto.Response{data=dto.BatchResult}
// @Router       /smart-records/batch-delete [post]
// @Security     BearerAuth
func (h *SmartRecordHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	response.OK(c, h.svc.BatchDelete(c.Request.Context(), req.IDs))
}

// ExportSelectedCSV 按选定 ID 列表导出智能记录 CSV。
//
// @Summary      导出选定智能记录 CSV
// @Description  根据勾选的记录 ID 列表导出智能记录为 CSV 文件
// @Tags         智能记录
// @Accept       json
// @Produce      text/csv
// @Param        body  body  dto.BatchIDsRequest  true  "记录 ID 列表"
// @Success      200   {file}    file    "CSV 文件"
// @Router       /smart-records/export-selected [post]
// @Security     BearerAuth
func (h *SmartRecordHandler) ExportSelectedCSV(c *gin.Context) {
	var req dto.BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	data, err := h.svc.ExportSelectedCSV(c.Request.Context(), req.IDs)
	if err != nil {
		attachError(c, err)
		return
	}
	writeCSV(c, "smart-records-selected.csv", data)
}
