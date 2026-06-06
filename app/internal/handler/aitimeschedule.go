package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// AITimeScheduleHandler 处理 AITimeSchedule HTTP 请求
type AITimeScheduleHandler struct {
	svc *service.AITimeScheduleService
}

// NewAITimeScheduleHandler 创建新的 AITimeScheduleHandler
func NewAITimeScheduleHandler(svc *service.AITimeScheduleService) *AITimeScheduleHandler {
	return &AITimeScheduleHandler{svc: svc}
}

// List 分页查询时间配置
//
// @Summary      时间配置列表
// @Description  分页查询时间配置
// @Tags         aitimeschedule
// @Produce      json
// @Param        page       query   int     false  "页码"      default(1)
// @Param        page_size  query   int     false  "每页数量"  default(20)
// @Param        keyword    query   string  false  "关键词搜索"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.AITimeSchedule}}
// @Router       /ai-time-schedules [get]
// @Security     BearerAuth
func (h *AITimeScheduleHandler) List(c *gin.Context) {
	var req dto.AITimeScheduleListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// ListAll 返回全部时间配置（供前端下拉选择）
//
// @Summary      全部时间配置
// @Description  返回全部时间配置，不分页
// @Tags         aitimeschedule
// @Produce      json
// @Success      200  {object}  dto.Response{data=[]model.AITimeSchedule}
// @Router       /ai-time-schedules/all [get]
// @Security     BearerAuth
func (h *AITimeScheduleHandler) ListAll(c *gin.Context) {
	items, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, items)
}

// Create 创建时间配置
//
// @Summary      创建时间配置
// @Description  创建时间配置
// @Tags         aitimeschedule
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateAITimeScheduleRequest  true  "时间配置信息"
// @Success      200   {object}  dto.Response{data=model.AITimeSchedule}
// @Router       /ai-time-schedules [post]
// @Security     BearerAuth
func (h *AITimeScheduleHandler) Create(c *gin.Context) {
	var req dto.CreateAITimeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	item, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, item)
}

// GetByID 查询单个时间配置
//
// @Summary      获取时间配置详情
// @Description  根据 ID 查询时间配置
// @Tags         aitimeschedule
// @Produce      json
// @Param        id   path   string  true  "ID"
// @Success      200  {object}  dto.Response{data=model.AITimeSchedule}
// @Router       /ai-time-schedules/{id} [get]
// @Security     BearerAuth
func (h *AITimeScheduleHandler) GetByID(c *gin.Context) {
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

// Update 更新时间配置
//
// @Summary      更新时间配置
// @Description  更新时间配置
// @Tags         aitimeschedule
// @Accept       json
// @Produce      json
// @Param        id    path   string                           true  "ID"
// @Param        body  body   dto.UpdateAITimeScheduleRequest  true  "时间配置信息"
// @Success      200   {object}  dto.Response
// @Router       /ai-time-schedules/{id} [put]
// @Security     BearerAuth
func (h *AITimeScheduleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "无效的 ID"))
		return
	}
	var req dto.UpdateAITimeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, req); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete 删除时间配置
//
// @Summary      删除时间配置
// @Description  删除时间配置
// @Tags         aitimeschedule
// @Produce      json
// @Param        id  path  string  true  "ID"
// @Success      200  {object}  dto.Response
// @Router       /ai-time-schedules/{id} [delete]
// @Security     BearerAuth
func (h *AITimeScheduleHandler) Delete(c *gin.Context) {
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
