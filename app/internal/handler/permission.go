package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// PermissionHandler handles HTTP requests for Permission operations.
type PermissionHandler struct {
	db *gorm.DB
}

// NewPermissionHandler creates a new PermissionHandler with the given database.
func NewPermissionHandler(db *gorm.DB) *PermissionHandler {
	return &PermissionHandler{db: db}
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
	var all []model.Permission
	if err := h.db.Order("sort_order ASC, created_at ASC").Find(&all).Error; err != nil {
		zap.L().Error("list permissions failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	tree := buildPermissionTree(all, nil)
	response.OK(c, tree)
}

// buildPermissionTree recursively builds a permission tree from a flat list.
func buildPermissionTree(all []model.Permission, parentID *string) []model.Permission {
	var result []model.Permission
	for _, p := range all {
		// Match nodes whose ParentID equals the given parentID (both nil for roots)
		if (parentID == nil && p.ParentID == nil) ||
			(parentID != nil && p.ParentID != nil && *parentID == *p.ParentID) {
			node := p
			node.Children = buildPermissionTree(all, &node.ID)
			result = append(result, node)
		}
	}
	return result
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

	// Check code uniqueness
	var count int64
	if err := h.db.Model(&model.Permission{}).Where("code = ?", req.Code).Count(&count).Error; err != nil {
		zap.L().Error("check permission code uniqueness failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}
	if count > 0 {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "权限编码已存在"))
		return
	}

	// Verify parent exists if specified
	if req.ParentID != nil {
		var parentCount int64
		if err := h.db.Model(&model.Permission{}).Where("id = ?", *req.ParentID).Count(&parentCount).Error; err != nil {
			zap.L().Error("check parent permission failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
			return
		}
		if parentCount == 0 {
			response.Err(c, apperrors.New(apperrors.ErrNotFound, "父级权限不存在"))
			return
		}
	}

	item := model.Permission{
		Name:      req.Name,
		Code:      req.Code,
		Path:      req.Path,
		Method:    req.Method,
		Type:      req.Type,
		ParentID:  req.ParentID,
		SortOrder: req.SortOrder,
	}

	if err := h.db.Create(&item).Error; err != nil {
		zap.L().Error("create permission failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.OK(c, item)
}
