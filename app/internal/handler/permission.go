package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
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
		response.Err(c, err)
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
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	perm, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, perm)
}
