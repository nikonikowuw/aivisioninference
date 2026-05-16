package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/middleware"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// UserHandler handles HTTP requests for User CRUD operations.
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler creates a new UserHandler with the given dependencies.
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// List returns a paginated list of users with optional search filters.
//
// @Summary      用户列表
// @Description  分页查询用户列表，支持按关键词、状态筛选
// @Tags         用户管理
// @Produce      json
// @Param        page       query   int     false  "页码"       default(1)
// @Param        page_size  query   int     false  "每页数量"   default(20)
// @Param        keyword    query   string  false  "关键词搜索（用户名/显示名/邮箱）"
// @Param        status     query   int     false  "状态筛选 (1=启用 0=禁用)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.User}}
// @Router       /users [get]
// @Security     BearerAuth
func (h *UserHandler) List(c *gin.Context) {
	var req dto.UserListRequest
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

// Create creates a new user with password hashing and optional role association.
//
// @Summary      创建用户
// @Description  创建新用户，密码自动加密，可关联角色
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateUserRequest  true  "用户信息"
// @Success      200   {object}  dto.Response{data=model.User}
// @Router       /users [post]
// @Security     BearerAuth
func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	user, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, user)
}

// GetByID returns a user by its ID.
//
// @Summary      获取用户详情
// @Description  根据 ID 查询用户信息
// @Tags         用户管理
// @Produce      json
// @Param        id   path   string  true  "用户 ID"
// @Success      200  {object}  dto.Response{data=model.User}
// @Router       /users/{id} [get]
// @Security     BearerAuth
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, user)
}

// Update updates an existing user by its ID.
//
// @Summary      更新用户
// @Description  更新用户信息，可更新角色关联
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id    path   string                    true  "用户 ID"
// @Param        body  body  dto.UpdateUserRequest      true  "用户信息"
// @Success      200   {object}  dto.Response
// @Router       /users/{id} [put]
// @Security     BearerAuth
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRootVal, _ := c.Get(middleware.ContextKeyIsRoot)
	isRoot, _ := isRootVal.(bool)
	if id == uid && req.Status != nil && *req.Status == 0 {
		attachError(c, apperrors.New(apperrors.ErrCannotDisableSelf, ""))
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, req, uid, isRoot); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete soft-deletes a user by its ID.
//
// @Summary      删除用户
// @Description  软删除用户
// @Tags         用户管理
// @Produce      json
// @Param        id  path  string  true  "用户 ID"
// @Success      200  {object}  dto.Response
// @Router       /users/{id} [delete]
// @Security     BearerAuth
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	currentUserID, _ := c.Get(middleware.ContextKeyUserID)
	uid, _ := currentUserID.(string)
	isRootVal, _ := c.Get(middleware.ContextKeyIsRoot)
	isRoot, _ := isRootVal.(bool)

	if err := h.svc.Delete(c.Request.Context(), id, uid, isRoot); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}
