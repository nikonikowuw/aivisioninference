package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// PermissionHandler handles HTTP requests for Permission operations.
type PermissionHandler struct {
	svc *service.PermissionService
}

// NewPermissionHandler creates a new PermissionHandler with the given dependencies.
func NewPermissionHandler(svc *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{svc: svc}
}

// Tree returns all permissions organized as a tree structure.
//
// @Summary      权限树
// @Description  返回所有权限的树形结构（按 sort_order 排序）
// @Tags         权限管理
// @Produce      json
// @Success      200  {object}  dto.Response{data=[]model.Permission}
// @Router       /permissions/tree [get]
// @Security     BearerAuth
func (h *PermissionHandler) Tree(c *gin.Context) {
	tree, err := h.svc.Tree(c.Request.Context())
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, tree)
}

// Create creates a new permission.
//
// @Summary      创建权限
// @Description  创建新权限（菜单或 API 权限）
// @Tags         权限管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreatePermissionRequest  true  "权限信息"
// @Success      200   {object}  dto.Response{data=model.Permission}
// @Router       /permissions [post]
// @Security     BearerAuth
func (h *PermissionHandler) Create(c *gin.Context) {
	var req dto.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	perm, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, perm)
}

// Update updates an existing permission.
//
// @Summary      更新权限
// @Description  更新权限信息（名称、编码、路径等）
// @Tags         权限管理
// @Accept       json
// @Produce      json
// @Param        id    path  string                       true  "权限 ID"
// @Param        body  body  dto.UpdatePermissionRequest  true  "权限信息"
// @Success      200   {object}  response.Response
// @Router       /permissions/{id} [put]
// @Security     BearerAuth
func (h *PermissionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, req); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete deletes a permission by its ID.
//
// @Summary      删除权限
// @Description  删除权限（如果已分配给角色则无法删除）
// @Tags         权限管理
// @Produce      json
// @Param        id  path  string  true  "权限 ID"
// @Success      200  {object}  response.Response
// @Router       /permissions/{id} [delete]
// @Security     BearerAuth
func (h *PermissionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}
