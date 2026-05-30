// Package handler 提供 HTTP 请求处理层（Controller），负责参数绑定、校验和响应返回。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// TaskHandler 处理后台任务管理相关的 HTTP 请求。
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler 创建一个新的 TaskHandler 实例。
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Create 创建一个新的后台任务（例如导出、清理等异步操作）。
//
// @Summary      创建任务
// @Description  创建新的后台任务
// @Tags         任务管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateTaskRequest  true  "任务信息"
// @Success      200   {object}  dto.Response{data=model.Task}
// @Router       /tasks [post]
// @Security     BearerAuth
func (h *TaskHandler) Create(c *gin.Context) {
	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	task, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, task)
}

// List 返回分页的任务列表，支持关键词、任务类型以及任务状态过滤。
//
// @Summary      任务列表
// @Description  分页查询任务列表，支持按关键词、类型、状态筛选
// @Tags         任务管理
// @Produce      json
// @Param        page      query   int     false  "页码"       default(1)
// @Param        page_size query   int     false  "每页数量"   default(20)
// @Param        keyword   query   string  false  "关键词搜索"
// @Param        type      query   string  false  "任务类型筛选"
// @Param        status    query   string  false  "任务状态筛选 (pending|running|completed|failed)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.Task}}
// @Router       /tasks [get]
// @Security     BearerAuth
func (h *TaskHandler) List(c *gin.Context) {
	var req dto.TaskListRequest
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

// GetByID 根据任务 ID 获取任务详情（包含错误信息或结果）。
//
// @Summary      获取任务详情
// @Description  根据 ID 查询任务信息
// @Tags         任务管理
// @Produce      json
// @Param        id   path   string  true  "任务 ID"
// @Success      200  {object}  dto.Response{data=model.Task}
// @Router       /tasks/{id} [get]
// @Security     BearerAuth
func (h *TaskHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	task, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, task)
}

// Cancel 取消一个排队中或运行中的任务。
//
// @Summary      取消任务
// @Description  取消尚未完成的任务
// @Tags         任务管理
// @Produce      json
// @Param        id  path  string  true  "任务 ID"
// @Success      200  {object}  dto.Response
// @Router       /tasks/{id}/cancel [post]
// @Security     BearerAuth
func (h *TaskHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Cancel(c.Request.Context(), id); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// BatchCancel 批量取消任务。
func (h *TaskHandler) BatchCancel(c *gin.Context) {
	var req dto.BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	response.OK(c, h.svc.BatchCancel(c.Request.Context(), req.IDs, currentLang(c)))
}

// ExportCSV 按当前筛选条件导出任务 CSV。
func (h *TaskHandler) ExportCSV(c *gin.Context) {
	var req dto.TaskListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	data, err := h.svc.ExportCSV(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	writeCSV(c, "tasks.csv", data)
}
