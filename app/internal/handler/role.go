package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// permCachePrefix must match the prefix used by the RBAC middleware.
const permCachePrefix = "perm:"

// RoleHandler handles HTTP requests for Role CRUD and permission assignment.
type RoleHandler struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewRoleHandler creates a new RoleHandler with the given database and Redis client.
func NewRoleHandler(db *gorm.DB, rdb *redis.Client) *RoleHandler {
	return &RoleHandler{db: db, rdb: rdb}
}

// List returns a paginated list of roles with optional search filters.
//
// @Summary      角色列表
// @Description  分页查询角色列表，支持按名称、描述筛选
// @Tags         角色管理
// @Produce      json
// @Param        page        query   int     false  "页码"       default(1)
// @Param        page_size   query   int     false  "每页数量"   default(20)
// @Param        name        query   string  false  "角色名称搜索"
// @Param        description query   string  false  "描述搜索"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.Role}}
// @Router       /roles [get]
// @Security     BearerAuth
func (h *RoleHandler) List(c *gin.Context) {
	var req dto.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	page := req.GetPage()
	pageSize := req.GetPageSize()

	query := h.db.Model(&model.Role{})
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if description := c.Query("description"); description != "" {
		query = query.Where("description LIKE ?", "%"+description+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		zap.L().Error("count roles failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	var items []model.Role
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("sort_order ASC, created_at DESC").Find(&items).Error; err != nil {
		zap.L().Error("list roles failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.Page(c, items, total, page, pageSize)
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
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	// Check name uniqueness
	var count int64
	if err := h.db.Model(&model.Role{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		zap.L().Error("check role name uniqueness failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}
	if count > 0 {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "角色名称已存在"))
		return
	}

	item := model.Role{
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		Status:      req.Status,
	}

	if err := h.db.Create(&item).Error; err != nil {
		zap.L().Error("create role failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.OK(c, item)
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
	var item model.Role
	if err := h.db.Where("id = ?", id).First(&item).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "角色不存在"))
		return
	}
	response.OK(c, item)
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
	var item model.Role
	if err := h.db.Where("id = ?", id).First(&item).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "角色不存在"))
		return
	}

	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	// Check name uniqueness if name is being changed
	if req.Name != "" && req.Name != item.Name {
		var count int64
		if err := h.db.Model(&model.Role{}).Where("name = ? AND id != ?", req.Name, id).Count(&count).Error; err != nil {
			zap.L().Error("check role name uniqueness failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
			return
		}
		if count > 0 {
			response.Err(c, apperrors.New(apperrors.ErrBadRequest, "角色名称已存在"))
			return
		}
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	updates["sort_order"] = req.SortOrder
	updates["status"] = req.Status

	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		zap.L().Error("update role failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
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

	// Check if role exists
	var item model.Role
	if err := h.db.Where("id = ?", id).First(&item).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "角色不存在"))
		return
	}

	// Check if role is assigned to any users
	var userCount int64
	if err := h.db.Table("user_roles").Where("role_id = ?", id).Count(&userCount).Error; err != nil {
		zap.L().Error("check role usage failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}
	if userCount > 0 {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "该角色已分配给用户，无法删除"))
		return
	}

	if err := h.db.Where("id = ?", id).Delete(&model.Role{}).Error; err != nil {
		zap.L().Error("delete role failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Invalidate all permission caches since role permissions changed
	h.invalidatePermCache()

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

	// Verify role exists
	var role model.Role
	if err := h.db.Where("id = ?", id).First(&role).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "角色不存在"))
		return
	}

	// Query permissions through the role_permissions join table
	var permissions []model.Permission
	if err := h.db.
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", id).
		Order("permissions.sort_order ASC, permissions.created_at ASC").
		Find(&permissions).Error; err != nil {
		zap.L().Error("get role permissions failed",
			zap.String("role_id", id),
			zap.Error(err),
		)
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
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

	// Verify role exists
	var role model.Role
	if err := h.db.Where("id = ?", id).First(&role).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "角色不存在"))
		return
	}

	var req dto.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	// Verify all permission IDs exist
	if len(req.PermissionIDs) > 0 {
		var count int64
		if err := h.db.Model(&model.Permission{}).
			Where("id IN ?", req.PermissionIDs).
			Count(&count).Error; err != nil {
			zap.L().Error("verify permission ids failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
			return
		}
		if int(count) != len(req.PermissionIDs) {
			response.Err(c, apperrors.New(apperrors.ErrBadRequest, "部分权限 ID 不存在"))
			return
		}
	}

	// Replace permissions in a transaction
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing associations
		if err := tx.Where("role_id = ?", id).Delete(&model.RolePermission{}).Error; err != nil {
			return fmt.Errorf("delete old role_permissions: %w", err)
		}

		// Insert new associations
		if len(req.PermissionIDs) > 0 {
			rolePerms := make([]model.RolePermission, 0, len(req.PermissionIDs))
			for _, pid := range req.PermissionIDs {
				rolePerms = append(rolePerms, model.RolePermission{
					RoleID:       id,
					PermissionID: pid,
				})
			}
			if err := tx.Create(&rolePerms).Error; err != nil {
				return fmt.Errorf("create role_permissions: %w", err)
			}
		}

		return nil
	}); err != nil {
		zap.L().Error("assign permissions failed",
			zap.String("role_id", id),
			zap.Error(err),
		)
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Invalidate all permission caches
	h.invalidatePermCache()

	response.OK(c, nil)
}

// invalidatePermCache removes all cached permission entries from Redis.
// This is called when role-permission assignments change.
func (h *RoleHandler) invalidatePermCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Scan for all perm:* keys and delete them
	var cursor uint64
	var deleted int64
	for {
		keys, nextCursor, err := h.rdb.Scan(ctx, cursor, permCachePrefix+"*", 100).Result()
		if err != nil {
			zap.L().Warn("scan perm cache keys failed", zap.Error(err))
			return
		}
		if len(keys) > 0 {
			if err := h.rdb.Del(ctx, keys...).Err(); err != nil {
				zap.L().Warn("delete perm cache keys failed", zap.Error(err))
			}
			deleted += int64(len(keys))
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if deleted > 0 {
		zap.L().Info("invalidated permission cache", zap.Int64("keys_deleted", deleted))
	}
}
