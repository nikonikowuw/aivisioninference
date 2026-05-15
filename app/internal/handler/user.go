package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// UserHandler handles HTTP requests for User CRUD operations.
type UserHandler struct {
	db *gorm.DB
}

// NewUserHandler creates a new UserHandler with the given database.
func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

// List returns a paginated list of users with optional search filters.
//
// @Summary      用户列表
// @Description  分页查询用户列表，支持按用户名、显示名、状态筛选
// @Tags         用户管理
// @Produce      json
// @Param        page         query   int     false  "页码"       default(1)
// @Param        page_size    query   int     false  "每页数量"   default(20)
// @Param        username     query   string  false  "用户名搜索"
// @Param        display_name query   string  false  "显示名搜索"
// @Param        status       query   int     false  "状态筛选 (1=启用 0=禁用)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.User}}
// @Router       /users [get]
// @Security     BearerAuth
func (h *UserHandler) List(c *gin.Context) {
	var req dto.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	page := req.GetPage()
	pageSize := req.GetPageSize()

	query := h.db.Model(&model.User{})
	if username := c.Query("username"); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if displayName := c.Query("display_name"); displayName != "" {
		query = query.Where("display_name LIKE ?", "%"+displayName+"%")
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.Atoi(statusStr); err == nil {
			query = query.Where("status = ?", status)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		zap.L().Error("count users failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	var items []model.User
	if err := query.Preload("Roles").Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		zap.L().Error("list users failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	response.Page(c, items, total, page, pageSize)
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
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	// Check username uniqueness
	var count int64
	if err := h.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		zap.L().Error("check username uniqueness failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}
	if count > 0 {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "用户名已存在"))
		return
	}

	// Hash password
	hashedPassword, err := hash.Hash(req.Password)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Build user
	user := model.User{
		Username:    req.Username,
		Password:    hashedPassword,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
		Status:      req.Status,
	}

	// Create user with roles in a transaction
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Associate roles
		if len(req.RoleIDs) > 0 {
			var roles []model.Role
			if err := tx.Where("id IN ?", req.RoleIDs).Find(&roles).Error; err != nil {
				return err
			}
			if err := tx.Model(&user).Association("Roles").Replace(roles); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		zap.L().Error("create user failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Reload with roles
	h.db.Preload("Roles").Where("id = ?", user.ID).First(&user)

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
	var item model.User
	if err := h.db.Preload("Roles").Where("id = ?", id).First(&item).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "用户不存在"))
		return
	}
	response.OK(c, item)
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
	var item model.User
	if err := h.db.Where("id = ?", id).First(&item).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "用户不存在"))
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	// Check username uniqueness if changed
	if req.Username != "" && req.Username != item.Username {
		var count int64
		if err := h.db.Model(&model.User{}).Where("username = ? AND id != ?", req.Username, id).Count(&count).Error; err != nil {
			zap.L().Error("check username uniqueness failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
			return
		}
		if count > 0 {
			response.Err(c, apperrors.New(apperrors.ErrBadRequest, "用户名已存在"))
			return
		}
	}

	updates := map[string]interface{}{}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = req.AvatarURL
	}
	updates["status"] = req.Status

	if err := h.db.Model(&item).Updates(updates).Error; err != nil {
		zap.L().Error("update user failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Update roles if provided
	if req.RoleIDs != nil {
		var roles []model.Role
		if err := h.db.Where("id IN ?", req.RoleIDs).Find(&roles).Error; err != nil {
			zap.L().Error("find roles failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
			return
		}
		if err := h.db.Model(&item).Association("Roles").Replace(roles); err != nil {
			zap.L().Error("update user roles failed", zap.Error(err))
			response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
			return
		}
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

	// Prevent deleting self
	userID, _ := c.Get("user_id")
	if uid, ok := userID.(string); ok && uid == id {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "不能删除当前登录用户"))
		return
	}

	if err := h.db.Where("id = ?", id).Delete(&model.User{}).Error; err != nil {
		zap.L().Error("delete user failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}
	response.OK(c, nil)
}
