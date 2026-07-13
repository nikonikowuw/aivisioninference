package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// EdgeNodeScheduledTaskHandler handles HTTP requests for scheduled tasks.
type EdgeNodeScheduledTaskHandler struct {
	svc *service.EdgeNodeScheduledTaskService
}

// NewEdgeNodeScheduledTaskHandler creates a new handler.
func NewEdgeNodeScheduledTaskHandler(svc *service.EdgeNodeScheduledTaskService) *EdgeNodeScheduledTaskHandler {
	return &EdgeNodeScheduledTaskHandler{svc: svc}
}

// Create creates a new scheduled task on a node.
//
// @Summary      创建定时任务
// @Description  为边缘节点创建 Shell 定时任务（一次性或 Cron）
// @Tags         边缘节点
// @Accept       json
// @Produce      json
// @Param        id    path  string                true  "节点 ID"
// @Param        body  body  dto.ScheduledTaskRequest  true  "任务信息"
// @Success      200   {object}  dto.Response
// @Router       /edge-nodes/{id}/scheduled-tasks [post]
// @Security     BearerAuth
func (h *EdgeNodeScheduledTaskHandler) Create(c *gin.Context) {
	id := c.Param("id")
	var req dto.ScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	task, err := h.svc.Create(c.Request.Context(), id, req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, task)
}

// List returns paginated scheduled tasks for a node.
//
// @Summary      定时任务列表
// @Description  分页查询边缘节点的定时任务
// @Tags         边缘节点
// @Produce      json
// @Param        id        path   int     false  "节点 ID"
// @Param        page      query  int     false  "页码"      default(1)
// @Param        page_size query  int     false  "每页数量"  default(20)
// @Param        status    query  string  false  "状态筛选(active/disabled)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.EdgeNodeScheduledTask}}
// @Router       /edge-nodes/{id}/scheduled-tasks [get]
// @Security     BearerAuth
func (h *EdgeNodeScheduledTaskHandler) List(c *gin.Context) {
	id := c.Param("id")
	var req dto.ScheduledTaskListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	tasks, total, err := h.svc.List(c.Request.Context(), id, req.GetPage(), req.GetPageSize(), req.Status)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, tasks, total, req.GetPage(), req.GetPageSize())
}

// GetByID returns a scheduled task by ID.
//
// @Summary      获取定时任务详情
// @Description  根据 ID 查询定时任务详情
// @Tags         边缘节点
// @Produce      json
// @Param        id       path  string  true  "任务 ID"
// @Success      200  {object}  dto.Response{data=model.EdgeNodeScheduledTask}
// @Router       /edge-nodes/{id}/scheduled-tasks/{task_id} [get]
// @Security     BearerAuth
func (h *EdgeNodeScheduledTaskHandler) GetByID(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		response.Err(c, apperrors.NewLocalized(apperrors.ErrBadRequest, "缺少任务 ID"))
		return
	}

	task, err := h.svc.GetByID(c.Request.Context(), taskID)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, task)
}

// Update updates a scheduled task.
//
// @Summary      更新定时任务
// @Description  更新边缘节点定时任务信息
// @Tags         边缘节点
// @Accept       json
// @Produce      json
// @Param        id       path  string                        true  "任务 ID"
// @Param        body     body  dto.ScheduledTaskUpdateRequest  true  "更新信息"
// @Success      200  {object}  dto.Response
// @Router       /edge-nodes/{id}/scheduled-tasks/{task_id} [put]
// @Security     BearerAuth
func (h *EdgeNodeScheduledTaskHandler) Update(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		response.Err(c, apperrors.NewLocalized(apperrors.ErrBadRequest, "缺少任务 ID"))
		return
	}

	var req dto.ScheduledTaskUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	task, err := h.svc.Update(c.Request.Context(), taskID, req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, task)
}

// Delete deletes a scheduled task.
//
// @Summary      删除定时任务
// @Description  删除边缘节点的定时任务
// @Tags         边缘节点
// @Produce      json
// @Param        id       path  string  true  "任务 ID"
// @Success      200  {object}  dto.Response
// @Router       /edge-nodes/{id}/scheduled-tasks/{task_id} [delete]
// @Security     BearerAuth
func (h *EdgeNodeScheduledTaskHandler) Delete(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		response.Err(c, apperrors.NewLocalized(apperrors.ErrBadRequest, "缺少任务 ID"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), taskID); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// ListExecutions returns execution records for a task.
//
// @Summary      任务执行记录
// @Description  查询定时任务的执行历史
// @Tags         边缘节点
// @Produce      json
// @Param        id        path   string  true  "节点 ID"
// @Param        task_id   query  string  true  "任务 ID"
// @Param        page      query  int     false "页码"      default(1)
// @Param        page_size query  int     false "每页数量"  default(20)
// @Param        status    query  string  false "状态筛选(running/success/failed/timeout)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.EdgeNodeTaskExecution}}
// @Router       /edge-nodes/{id}/task-executions [get]
// @Security     BearerAuth
func (h *EdgeNodeScheduledTaskHandler) ListExecutions(c *gin.Context) {
	var req dto.TaskExecutionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	execs, total, err := h.svc.ListExecutions(c.Request.Context(), req.TaskID, req.GetPage(), req.GetPageSize(), req.Status)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, execs, total, req.GetPage(), req.GetPageSize())
}

// HandleCallback receives execution results from the engine via HTTP callback.
//
// @Summary      任务执行回调
// @Description  引擎执行结果回调（引擎通过 HTTP POST 调用此接口）
// @Tags         边缘节点
// @Accept       json
// @Produce      json
// @Param        id    path  string                           true  "节点 ID"
// @Param        body  body  dto.TaskExecutionCallbackRequest true  "执行结果"
// @Success      200   {object}  dto.Response
// @Router       /edge-nodes/{id}/task-executions/callback [post]
func (h *EdgeNodeScheduledTaskHandler) HandleCallback(c *gin.Context) {
	id := c.Param("id")
	var req dto.TaskExecutionCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.svc.HandleExecutionCallback(c.Request.Context(), id, req); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}
