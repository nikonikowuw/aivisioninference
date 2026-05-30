// Package handler 提供 HTTP 请求处理层（Controller），负责参数绑定、校验和响应返回。
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

const maxCSVImportSize = 10 * 1024 * 1024

// UserHandler 处理用户管理相关的 HTTP 请求（增删改查、密码重置及头像上传）。
type UserHandler struct {
	svc     *service.UserService
	authSvc *service.AuthService
}

// NewUserHandler 创建一个新的 UserHandler 实例。
func NewUserHandler(svc *service.UserService, authSvc *service.AuthService) *UserHandler {
	return &UserHandler{svc: svc, authSvc: authSvc}
}

// List 返回分页的用户列表，支持关键字（用户名/显示名/邮箱）搜索和启用状态筛选。
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
		attachError(c, badRequestError(c, err))
		return
	}

	// 获取分页数据
	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// Create 创建一个新的用户，自动哈希化密码，并可选地关联角色。
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
		attachError(c, badRequestError(c, err))
		return
	}

	// 提交服务层创建
	user, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, user)
}

// GetByID 根据用户 ID 获取用户的元数据信息。
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

// Update 更新用户的基本属性和角色关联，执行层级防越权判定，且不允许自己禁用自己。
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
		attachError(c, badRequestError(c, err))
		return
	}

	uid, isRoot := currentUserContext(c)

	// 安全校验：禁止用户将自身的账号状态设置为禁用
	if id == uid && req.Status != nil && *req.Status == 0 {
		attachError(c, apperrors.New(apperrors.ErrCannotDisableSelf, ""))
		return
	}

	// 提交更新
	if err := h.svc.Update(c.Request.Context(), id, req, uid, isRoot); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete 软删除用户，执行层级防越权判定。
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
	uid, isRoot := currentUserContext(c)

	// 执行软删除
	if err := h.svc.Delete(c.Request.Context(), id, uid, isRoot); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// BatchDelete 批量软删除用户，逐条复用单条删除的层级与自删除保护。
//
// @Summary      批量删除用户
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.BatchIDsRequest  true  "用户 ID 列表"
// @Success      200   {object}  dto.Response{data=dto.BatchResult}
// @Router       /users/batch-delete [post]
// @Security     BearerAuth
func (h *UserHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	uid, isRoot := currentUserContext(c)
	response.OK(c, h.svc.BatchDelete(c.Request.Context(), req.IDs, uid, isRoot, currentLang(c)))
}

// BatchUpdateStatus 批量启用或禁用用户，保留单条更新的权限校验。
//
// @Summary      批量更新用户状态
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.BatchUpdateUserStatusRequest  true  "用户状态"
// @Success      200   {object}  dto.Response{data=dto.BatchResult}
// @Router       /users/batch-status [put]
// @Security     BearerAuth
func (h *UserHandler) BatchUpdateStatus(c *gin.Context) {
	var req dto.BatchUpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	uid, isRoot := currentUserContext(c)
	response.OK(c, h.svc.BatchUpdateStatus(c.Request.Context(), req.IDs, req.Status, uid, isRoot, currentLang(c)))
}

// ExportCSV 按当前筛选条件导出用户 CSV。
//
// @Summary      导出用户 CSV
// @Tags         用户管理
// @Produce      text/csv
// @Success      200  {file}  file
// @Router       /users/export [get]
// @Security     BearerAuth
func (h *UserHandler) ExportCSV(c *gin.Context) {
	var req dto.UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	data, err := h.svc.ExportCSV(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	writeCSV(c, "users.csv", data)
}

// ImportCSV 从 CSV 文件批量导入用户。
//
// @Summary      导入用户 CSV
// @Tags         用户管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "CSV 文件"
// @Success      200   {object}  dto.Response{data=dto.BatchResult}
// @Router       /users/import [post]
// @Security     BearerAuth
func (h *UserHandler) ImportCSV(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}
	if fileHeader.Size > maxCSVImportSize {
		attachError(c, apperrors.New(apperrors.ErrFileTooLarge, ""))
		return
	}
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".csv") {
		attachError(c, apperrors.New(apperrors.ErrFileInvalidType, ""))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}
	defer file.Close()
	result, err := h.svc.ImportCSV(c.Request.Context(), file, currentLang(c))
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, result)
}

// ResetPassword 管理员强制重置某个用户的密码（无需提供旧密码），执行层级防越权校验。
//
// @Summary      重置用户密码
// @Description  管理员重置指定用户的密码（不需要旧密码）
// @Tags         用户管理
// @Accept       json
// @Produce      json
// @Param        id    path   string                    true  "用户 ID"
// @Param        body  body   dto.ResetPasswordRequest  true  "新密码"
// @Success      200   {object}  dto.Response
// @Failure      400   {object}  dto.Response
// @Failure      403   {object}  dto.Response
// @Failure      404   {object}  dto.Response
// @Router       /users/{id}/password [put]
// @Security     BearerAuth
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	uid, isRoot := currentUserContext(c)

	// 执行密码重置
	if err := h.svc.ResetPassword(c.Request.Context(), id, req.Password, uid, isRoot); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// UploadAvatar 管理员为指定 ID 的用户上传头像文件。
//
// @Summary      管理员上传用户头像
// @Description  管理员为指定用户上传头像
// @Tags         用户管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        id      path      string  true  "用户 ID"
// @Param        avatar  formData  file    true  "头像文件"
// @Success      200     {object}  dto.Response{data=dto.AvatarUploadResponse}
// @Failure      400     {object}  dto.Response
// @Failure      403     {object}  dto.Response
// @Router       /users/{id}/avatar [post]
// @Security     BearerAuth
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	id := c.Param("id")

	// 提取文件分片/文件流
	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, "缺少头像文件"))
		return
	}

	// 调用认证服务上传并绑定头像元数据
	avatarURL, err := h.authSvc.UploadAvatar(c.Request.Context(), id, fileHeader)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, dto.AvatarUploadResponse{AvatarURL: avatarURL})
}
