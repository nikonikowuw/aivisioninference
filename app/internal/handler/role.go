// Package handler 提供 HTTP 请求处理层（Controller），负责参数绑定、校验和响应返回。
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// RoleHandler 处理角色管理相关的 HTTP 请求（增删改查及权限分配）。
type RoleHandler struct {
	svc *service.RoleService
}

// NewRoleHandler 创建一个新的 RoleHandler 实例。
func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// List 返回分页的角色列表，支持关键词（名称/描述）搜索。
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
		attachError(c, badRequestError(c, err))
		return
	}

	// 查找符合过滤条件的角色列表
	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// Create 创建一个新的角色，验证创建者的层级关系以防越权。
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
		attachError(c, badRequestError(c, err))
		return
	}

	// 获取当前操作人的用户 ID 及其超级管理员标识 (Root) 用于层级权限判定
	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRoot, _ := c.Get(middleware.ContextKeyIsRoot)
	root, _ := isRoot.(bool)

	// 创建角色
	role, err := h.svc.Create(c.Request.Context(), req, uid, root)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, role)
}

// GetByID 根据角色 ID 获取角色的元数据。
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

// Update 更新已存在的某个角色的属性信息，并校验层级操作权限。
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
		attachError(c, badRequestError(c, err))
		return
	}

	// 获取操作人信息用于防越权校验
	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRoot, _ := c.Get(middleware.ContextKeyIsRoot)
	root, _ := isRoot.(bool)

	// 执行更新
	if err := h.svc.Update(c.Request.Context(), id, req, uid, root); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete 软删除角色，并校验层级操作权限。
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

	// 获取操作人信息用于防越权校验
	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRoot, _ := c.Get(middleware.ContextKeyIsRoot)
	root, _ := isRoot.(bool)

	// 执行软删除
	if err := h.svc.Delete(c.Request.Context(), id, uid, root); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// GetPermissions 获取指定角色已绑定的所有权限列表。
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

// AssignPermissions 全量覆盖更新指定角色的权限绑定关系，执行层级防越权校验。
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
		attachError(c, badRequestError(c, err))
		return
	}

	// 获取操作人信息用于越权检测
	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRoot, _ := c.Get(middleware.ContextKeyIsRoot)
	root, _ := isRoot.(bool)

	// 进行角色授权
	if err := h.svc.AssignPermissions(c.Request.Context(), id, req, uid, root); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}
