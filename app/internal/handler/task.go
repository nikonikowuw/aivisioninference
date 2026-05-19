package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// TaskHandler handles HTTP requests for background task management.
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler creates a new TaskHandler with the given dependencies.
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Create creates a new background task record.
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

// List returns a paginated list of tasks with optional filters.
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

// GetByID returns a task by its ID.
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

// Cancel cancels a pending or running task.
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
