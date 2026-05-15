package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// TaskHandler handles HTTP requests for background task management.
type TaskHandler struct {
	db *gorm.DB
}

// NewTaskHandler creates a new TaskHandler with the given database.
func NewTaskHandler(db *gorm.DB) *TaskHandler {
	return &TaskHandler{db: db}
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
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	item := model.Task{
		TaskID:     uuid.New().String(),
		Type:       req.Type,
		Payload:    req.Payload,
		Status:     "pending",
		MaxRetries: 3,
	}

	if err := h.db.Create(&item).Error; err != nil {
		zap.L().Error("create task failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.OK(c, item)
}

// List returns a paginated list of tasks with optional filters.
//
// @Summary      任务列表
// @Description  分页查询任务列表，支持按类型、状态筛选
// @Tags         任务管理
// @Produce      json
// @Param        page      query   int     false  "页码"       default(1)
// @Param        page_size query   int     false  "每页数量"   default(20)
// @Param        type      query   string  false  "任务类型筛选"
// @Param        status    query   string  false  "任务状态筛选 (pending|running|completed|failed)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.Task}}
// @Router       /tasks [get]
// @Security     BearerAuth
func (h *TaskHandler) List(c *gin.Context) {
	var req dto.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	page := req.GetPage()
	pageSize := req.GetPageSize()

	query := h.db.Model(&model.Task{})
	if taskType := c.Query("type"); taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		zap.L().Error("count tasks failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	var items []model.Task
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		zap.L().Error("list tasks failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.Page(c, items, total, page, pageSize)
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
	var item model.Task
	if err := h.db.Where("id = ?", id).First(&item).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "任务不存在"))
		return
	}
	response.OK(c, item)
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

	var item model.Task
	if err := h.db.Where("id = ?", id).First(&item).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "任务不存在"))
		return
	}

	// Only allow cancelling pending or running tasks
	if item.Status != "pending" && item.Status != "running" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "任务状态不允许取消"))
		return
	}

	now := time.Now()
	if err := h.db.Model(&item).Updates(map[string]interface{}{
		"status":      "cancelled",
		"finished_at": &now,
	}).Error; err != nil {
		zap.L().Error("cancel task failed",
			zap.String("task_id", id),
			zap.Error(err),
		)
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.OK(c, nil)
}
