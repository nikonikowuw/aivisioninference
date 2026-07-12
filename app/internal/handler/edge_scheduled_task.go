package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// EdgeScheduledTaskHandler 处理计划任务 HTTP 请求
type EdgeScheduledTaskHandler struct {
	svc *service.EdgeScheduledTaskService
}

// NewEdgeScheduledTaskHandler 创建新的 EdgeScheduledTaskHandler
func NewEdgeScheduledTaskHandler(svc *service.EdgeScheduledTaskService) *EdgeScheduledTaskHandler {
	return &EdgeScheduledTaskHandler{svc: svc}
}

// List 分页查询计划任务列表
//
// @Summary      计划任务列表
// @Description  分页查询计划任务列表
// @Tags         计划任务
// @Produce      json
// @Param        page       query   int     false  "页码"      default(1)
// @Param        page_size  query   int     false  "每页数量"  default(20)
// @Param        keyword    query   string  false  "关键词搜索"
// @Param        enabled    query   bool    false  "启用状态筛选"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.EdgeScheduledTask}}
// @Router       /edge-scheduled-tasks [get]
// @Security     BearerAuth
func (h *EdgeScheduledTaskHandler) List(c *gin.Context) {
	var req dto.EdgeScheduledTaskListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// Create 创建计划任务
//
// @Summary      创建计划任务
// @Description  创建定时向边缘节点下发命令的计划任务
// @Tags         计划任务
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateEdgeScheduledTaskRequest  true  "计划任务信息"
// @Success      200   {object}  dto.Response{data=model.EdgeScheduledTask}
// @Router       /edge-scheduled-tasks [post]
// @Security     BearerAuth
func (h *EdgeScheduledTaskHandler) Create(c *gin.Context) {
	var req dto.CreateEdgeScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	item, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, item)
}

// GetByID 查询单个计划任务
//
// @Summary      获取计划任务详情
// @Description  根据 ID 查询计划任务
// @Tags         计划任务
// @Produce      json
// @Param        id   path   string  true  "ID"
// @Success      200  {object}  dto.Response{data=model.EdgeScheduledTask}
// @Router       /edge-scheduled-tasks/{id} [get]
// @Security     BearerAuth
func (h *EdgeScheduledTaskHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "无效的 ID"))
		return
	}
	item, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, item)
}

// Update 更新计划任务
//
// @Summary      更新计划任务
// @Description  更新计划任务配置
// @Tags         计划任务
// @Accept       json
// @Produce      json
// @Param        id    path   string                           true  "ID"
// @Param        body  body   dto.UpdateEdgeScheduledTaskRequest  true  "计划任务信息"
// @Success      200   {object}  dto.Response
// @Router       /edge-scheduled-tasks/{id} [put]
// @Security     BearerAuth
func (h *EdgeScheduledTaskHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "无效的 ID"))
		return
	}
	var req dto.UpdateEdgeScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, req); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete 删除计划任务
//
// @Summary      删除计划任务
// @Description  删除计划任务
// @Tags         计划任务
// @Produce      json
// @Param        id  path  string  true  "ID"
// @Success      200  {object}  dto.Response
// @Router       /edge-scheduled-tasks/{id} [delete]
// @Security     BearerAuth
func (h *EdgeScheduledTaskHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "无效的 ID"))
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, nil)
}

// ToggleEnabled 启用/禁用计划任务
//
// @Summary      切换计划任务启用状态
// @Description  启用或禁用指定的计划任务
// @Tags         计划任务
// @Produce      json
// @Param        id  path  string  true  "ID"
// @Success      200  {object}  dto.Response{data=model.EdgeScheduledTask}
// @Router       /edge-scheduled-tasks/{id}/toggle [put]
// @Security     BearerAuth
func (h *EdgeScheduledTaskHandler) ToggleEnabled(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "无效的 ID"))
		return
	}
	item, err := h.svc.ToggleEnabled(c.Request.Context(), id)
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, item)
}

// ListRecords 查询执行记录
//
// @Summary      计划任务执行记录
// @Description  分页查询计划任务执行记录
// @Tags         计划任务
// @Produce      json
// @Param        page       query   int     false  "页码"      default(1)
// @Param        page_size  query   int     false  "每页数量"  default(20)
// @Param        task_id    query   string  false  "任务 ID 筛选"
// @Param        status     query   string  false  "状态筛选(pending/dispatched/success/failed)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.EdgeScheduledTaskRecord}}
// @Router       /edge-scheduled-tasks/records [get]
// @Security     BearerAuth
func (h *EdgeScheduledTaskHandler) ListRecords(c *gin.Context) {
	var req dto.EdgeScheduledTaskRecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	items, total, err := h.svc.ListRecords(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// RetryRecord 重试失败的执行记录
//
// @Summary      重试任务执行
// @Description  重试失败的计划任务执行记录
// @Tags         计划任务
// @Produce      json
// @Param        id          path   string  true  "任务 ID"
// @Param        record_id   path   string  true  "执行记录 ID"
// @Success      200  {object}  dto.Response{data=model.EdgeScheduledTaskRecord}
// @Router       /edge-scheduled-tasks/{id}/records/{record_id}/retry [post]
// @Security     BearerAuth
func (h *EdgeScheduledTaskHandler) RetryRecord(c *gin.Context) {
	recordID := c.Param("record_id")
	if recordID == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "无效的执行记录 ID"))
		return
	}

	record, err := h.svc.RetryRecord(c.Request.Context(), recordID)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, record)
}
