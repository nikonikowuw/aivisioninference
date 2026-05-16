package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/middleware"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// RoleHandler handles HTTP requests for Role CRUD and permission assignment.
type RoleHandler struct {
	svc *service.RoleService
}

// NewRoleHandler creates a new RoleHandler with the given dependencies.
func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// List returns a paginated list of roles with optional search filters.
//
// @Summary      角色列表
// @Description  分页查询角色列表，支持关键词搜索
// @Tags         角色管理
// @Produce      json
// @Param        page       query   int     false  "页码"       default(1)
// @Param        page_size  query   int     false  "每页数量"   default(20)
// @Param        keyword    query   string  false  "关键词搜索（名称/描述）"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.Role}}
// @Router       /roles [get]
// @Security     BearerAuth
func (h *RoleHandler) List(c *gin.Context) {
	var req dto.RoleListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// Create creates a new role.
//
// @Summary      创建角色
// @Description  创建新角色
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateRoleRequest  true  "角色信息"
// @Success      200   {object}  dto.Response{data=model.Role}
// @Router       /roles [post]
// @Security     BearerAuth
func (h *RoleHandler) Create(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRoot, _ := c.Get(middleware.ContextKeyIsRoot)
	root, _ := isRoot.(bool)

	role, err := h.svc.Create(c.Request.Context(), req, uid, root)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, role)
}

// GetByID returns a role by its ID.
//
// @Summary      获取角色详情
// @Description  根据 ID 查询角色信息
// @Tags         角色管理
// @Produce      json
// @Param        id   path   string  true  "角色 ID"
// @Success      200  {object}  dto.Response{data=model.Role}
// @Router       /roles/{id} [get]
// @Security     BearerAuth
func (h *RoleHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	role, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, role)
}

// Update updates an existing role by its ID.
//
// @Summary      更新角色
// @Description  更新角色信息
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Param        id    path   string                    true  "角色 ID"
// @Param        body  body  dto.UpdateRoleRequest      true  "角色信息"
// @Success      200   {object}  dto.Response
// @Router       /roles/{id} [put]
// @Security     BearerAuth
func (h *RoleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRoot, _ := c.Get(middleware.ContextKeyIsRoot)
	root, _ := isRoot.(bool)

	if err := h.svc.Update(c.Request.Context(), id, req, uid, root); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete soft-deletes a role by its ID.
//
// @Summary      删除角色
// @Description  删除角色
// @Tags         角色管理
// @Produce      json
// @Param        id  path  string  true  "角色 ID"
// @Success      200  {object}  dto.Response
// @Router       /roles/{id} [delete]
// @Security     BearerAuth
func (h *RoleHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRoot, _ := c.Get(middleware.ContextKeyIsRoot)
	root, _ := isRoot.(bool)

	if err := h.svc.Delete(c.Request.Context(), id, uid, root); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// GetPermissions returns the permissions assigned to a role.
//
// @Summary      获取角色权限
// @Description  根据角色 ID 查询已分配的权限列表
// @Tags         角色管理
// @Produce      json
// @Param        id   path   string  true  "角色 ID"
// @Success      200  {object}  dto.Response{data=[]model.Permission}
// @Router       /roles/{id}/permissions [get]
// @Security     BearerAuth
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	id := c.Param("id")
	permissions, err := h.svc.GetPermissions(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, permissions)
}

// AssignPermissions replaces all permissions of a role with the given set.
//
// @Summary      分配权限
// @Description  替换角色的全部权限（全量覆盖）
// @Tags         角色管理
// @Accept       json
// @Produce      json
// @Param        id    path   string                         true  "角色 ID"
// @Param        body  body   dto.AssignPermissionsRequest   true  "权限 ID 列表"
// @Success      200   {object}  dto.Response
// @Router       /roles/{id}/permissions [put]
// @Security     BearerAuth
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id := c.Param("id")
	var req dto.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRoot, _ := c.Get(middleware.ContextKeyIsRoot)
	root, _ := isRoot.(bool)

	if err := h.svc.AssignPermissions(c.Request.Context(), id, req, uid, root); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}
