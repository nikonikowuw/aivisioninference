package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// EdgeNodeTagHandler 处理边缘节点标签 HTTP 请求
type EdgeNodeTagHandler struct {
	svc *service.EdgeNodeTagService
}

// NewEdgeNodeTagHandler 创建新的 EdgeNodeTagHandler
func NewEdgeNodeTagHandler(svc *service.EdgeNodeTagService) *EdgeNodeTagHandler {
	return &EdgeNodeTagHandler{svc: svc}
}

// List 分页查询标签列表
//
// @Summary      标签列表
// @Description  分页查询标签列表
// @Tags         节点标签
// @Produce      json
// @Param        page       query   int     false  "页码"      default(1)
// @Param        page_size  query   int     false  "每页数量"  default(20)
// @Param        keyword    query   string  false  "关键词搜索"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.EdgeNodeTag}}
// @Router       /edge-node-tags [get]
// @Security     BearerAuth
func (h *EdgeNodeTagHandler) List(c *gin.Context) {
	var req dto.EdgeNodeTagListRequest
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

// ListAll 返回全部标签（供前端下拉选择）
//
// @Summary      全部标签
// @Description  返回全部标签，不分页
// @Tags         节点标签
// @Produce      json
// @Success      200  {object}  dto.Response{data=[]model.EdgeNodeTag}
// @Router       /edge-node-tags/all [get]
// @Security     BearerAuth
func (h *EdgeNodeTagHandler) ListAll(c *gin.Context) {
	items, err := h.svc.ListAll(c.Request.Context())
	if err != nil {
		response.Err(c, err)
		return
	}
	response.OK(c, items)
}

// Create 创建标签
//
// @Summary      创建标签
// @Description  创建标签
// @Tags         节点标签
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateEdgeNodeTagRequest  true  "标签信息"
// @Success      200   {object}  dto.Response{data=model.EdgeNodeTag}
// @Router       /edge-node-tags [post]
// @Security     BearerAuth
func (h *EdgeNodeTagHandler) Create(c *gin.Context) {
	var req dto.CreateEdgeNodeTagRequest
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

// GetByID 查询单个标签
//
// @Summary      获取标签详情
// @Description  根据 ID 查询标签
// @Tags         节点标签
// @Produce      json
// @Param        id   path   string  true  "ID"
// @Success      200  {object}  dto.Response{data=model.EdgeNodeTag}
// @Router       /edge-node-tags/{id} [get]
// @Security     BearerAuth
func (h *EdgeNodeTagHandler) GetByID(c *gin.Context) {
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

// Update 更新标签
//
// @Summary      更新标签
// @Description  更新标签
// @Tags         节点标签
// @Accept       json
// @Produce      json
// @Param        id    path   string                           true  "ID"
// @Param        body  body   dto.UpdateEdgeNodeTagRequest  true  "标签信息"
// @Success      200   {object}  dto.Response
// @Router       /edge-node-tags/{id} [put]
// @Security     BearerAuth
func (h *EdgeNodeTagHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "无效的 ID"))
		return
	}
	var req dto.UpdateEdgeNodeTagRequest
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

// Delete 删除标签
//
// @Summary      删除标签
// @Description  删除标签
// @Tags         节点标签
// @Produce      json
// @Param        id  path  string  true  "ID"
// @Success      200  {object}  dto.Response
// @Router       /edge-node-tags/{id} [delete]
// @Security     BearerAuth
func (h *EdgeNodeTagHandler) Delete(c *gin.Context) {
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
