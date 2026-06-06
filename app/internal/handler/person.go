// Package handler 提供人员管理模块的 HTTP 请求处理层。
package handler

import (
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// imageMimeTypes 文件扩展名到 MIME 类型的映射，避免每次请求重复分配。
var imageMimeTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
}

// PersonHandler 处理人员管理相关的 HTTP 请求。
type PersonHandler struct {
	svc *service.PersonService
}

// NewPersonHandler 创建 PersonHandler。
func NewPersonHandler(svc *service.PersonService) *PersonHandler {
	return &PersonHandler{svc: svc}
}

// List 返回人员分页列表。
// @Summary      人员列表
// @Description  分页查询人员列表
// @Tags         人员管理
// @Produce      json
// @Param        page             query   int     false  "页码"       default(1)
// @Param        page_size        query   int     false  "每页数量"   default(20)
// @Param        keyword          query   string  false  "关键词"
// @Param        group_id         query   string  false  "分组ID"
// @Param        tag_id           query   string  false  "标签ID"
// @Param        embedding_status query   string  false  "特征状态"
// @Param        enabled          query   bool    false  "启用状态"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.PersonResponse}}
// @Router       /persons [get]
// @Security     BearerAuth
func (h *PersonHandler) List(c *gin.Context) {
	var req dto.PersonListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// GetByID 获取人员详情。
// @Summary      人员详情
// @Description  根据 ID 查询人员信息
// @Tags         人员管理
// @Produce      json
// @Param        id   path   string  true  "人员 ID"
// @Success      200  {object}  dto.Response{data=dto.PersonResponse}
// @Router       /persons/{id} [get]
// @Security     BearerAuth
func (h *PersonHandler) GetByID(c *gin.Context) {
	item, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}

// Create 创建人员。
// @Summary      创建人员
// @Description  创建人员记录并上传人脸图片
// @Tags         人员管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        image      formData  file    true   "人脸图片"
// @Param        person_name formData  string true   "姓名"
// @Param        person_code formData  string false  "人员编号"
// @Param        gender     formData  string  false  "性别"
// @Param        phone      formData  string  false  "手机号"
// @Param        id_number  formData  string  false  "证件号"
// @Param        remark     formData  string  false  "备注"
// @Param        group_ids  formData  string  false  "分组ID列表(JSON数组)"
// @Param        tag_ids    formData  string  false  "标签ID列表(JSON数组)"
// @Success      200  {object}  dto.Response{data=dto.PersonResponse}
// @Router       /persons [post]
// @Security     BearerAuth
func (h *PersonHandler) Create(c *gin.Context) {
	var req dto.PersonCreateRequest
	if err := c.ShouldBind(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	fileHeader, _ := c.FormFile("image")
	item, err := h.svc.Create(c.Request.Context(), req, fileHeader)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}

// Update 更新人员。
// @Summary      更新人员
// @Description  更新人员信息，可更换图片
// @Tags         人员管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        id         path    string  true   "人员 ID"
// @Param        image      formData  file    false  "人脸图片"
// @Param        person_name formData  string false  "姓名"
// @Param        person_code formData  string false  "人员编号"
// @Param        gender     formData  string  false  "性别"
// @Param        phone      formData  string  false  "手机号"
// @Param        id_number  formData  string  false  "证件号"
// @Param        remark     formData  string  false  "备注"
// @Param        group_ids  formData  string  false  "分组ID列表(JSON数组)"
// @Param        tag_ids    formData  string  false  "标签ID列表(JSON数组)"
// @Success      200  {object}  dto.Response{data=dto.PersonResponse}
// @Router       /persons/{id} [put]
// @Security     BearerAuth
func (h *PersonHandler) Update(c *gin.Context) {
	var req dto.PersonUpdateRequest
	if err := c.ShouldBind(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	fileHeader, _ := c.FormFile("image")
	item, err := h.svc.Update(c.Request.Context(), c.Param("id"), req, fileHeader)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}

// Delete 删除人员。
// @Summary      删除人员
// @Tags         人员管理
// @Produce      json
// @Param        id  path  string  true  "人员 ID"
// @Success      200  {object}  dto.Response
// @Router       /persons/{id} [delete]
// @Security     BearerAuth
func (h *PersonHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// BatchDelete 批量删除。
// @Summary      批量删除人员
// @Tags         人员管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.BatchIDsRequest  true  "ID列表"
// @Success      200   {object}  dto.Response
// @Router       /persons/batch-delete [post]
// @Security     BearerAuth
func (h *PersonHandler) BatchDelete(c *gin.Context) {
	var req dto.BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	if err := h.svc.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// BatchToggle 批量启用/禁用。
// @Summary      批量启用/禁用人员
// @Tags         人员管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.PersonToggleRequest  true  "启用/禁用请求"
// @Success      200   {object}  dto.Response
// @Router       /persons/batch-toggle [post]
// @Security     BearerAuth
func (h *PersonHandler) BatchToggle(c *gin.Context) {
	var req dto.PersonToggleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	if err := h.svc.BatchToggle(c.Request.Context(), req.IDs, req.Enabled); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// RetryEmbedding 重提特征。
// @Summary      重提特征
// @Tags         人员管理
// @Produce      json
// @Param        id  path  string  true  "人员 ID"
// @Success      200  {object}  dto.Response
// @Router       /persons/{id}/retry-embedding [post]
// @Security     BearerAuth
func (h *PersonHandler) RetryEmbedding(c *gin.Context) {
	if err := h.svc.RetryEmbedding(c.Request.Context(), c.Param("id")); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// BatchRetryEmbedding 批量重提特征。
// @Summary      批量重提特征
// @Tags         人员管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.BatchIDsRequest  true  "ID列表"
// @Success      200   {object}  dto.Response{data=service.BatchRetryResult}
// @Router       /persons/batch-retry-embedding [post]
// @Security     BearerAuth
func (h *PersonHandler) BatchRetryEmbedding(c *gin.Context) {
	var req dto.BatchIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	result, err := h.svc.BatchRetryEmbedding(c.Request.Context(), req.IDs)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, result)
}

// ExportExcel 导出人员 Excel。
// @Summary      导出人员Excel
// @Tags         人员管理
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param        keyword  query  string  false  "关键词"
// @Success      200  {file}  file
// @Router       /persons/export [get]
// @Security     BearerAuth
func (h *PersonHandler) ExportExcel(c *gin.Context) {
	var req dto.PersonListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="persons.xlsx"`)
	// 采用流式写入避免大表导出在内存中聚集。
	if err := h.svc.ExportExcel(c.Request.Context(), req, c.Writer); err != nil {
		attachError(c, err)
		return
	}
}

// ViewImage 查看人员图片。
// @Summary      查看图片
// @Description  根据文件名流式返回图片内容
// @Tags         人员管理
// @Param        filename  path  string  true  "图片文件名"
// @Success      200  {file}  image
// @Router       /persons/image/{filename} [get]
// @Security     BearerAuth
func (h *PersonHandler) ViewImage(c *gin.Context) {
	filename := c.Param("filename")
	// 路径遍历防护：禁止包含 .. 或 / 的文件名
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}
	// 仅允许安全的图片扩展名
	ext := strings.ToLower(filepath.Ext(filename))
	contentType, ok := imageMimeTypes[ext]
	if !ok {
		attachError(c, apperrors.New(apperrors.ErrFileInvalidType, ""))
		return
	}
	// 从 Service/Storage 获取图片流
	rc, err := h.svc.GetImage(c.Request.Context(), "persons/"+filename)
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrNotFound, ""))
		return
	}
	defer rc.Close()

	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("Content-Type", contentType)
	_, _ = io.Copy(c.Writer, rc)
}

// ListGroups 查询分组列表。
// @Summary      分组列表
// @Tags         人员分组
// @Produce      json
// @Success      200  {object}  dto.Response{data=[]dto.PersonGroupResponse}
// @Router       /person-groups [get]
// @Security     BearerAuth
func (h *PersonHandler) ListGroups(c *gin.Context) {
	items, err := h.svc.ListGroups(c.Request.Context())
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, items)
}

// CreateGroup 创建分组。
// @Summary      创建分组
// @Tags         人员分组
// @Accept       json
// @Produce      json
// @Param        body  body  dto.PersonGroupRequest  true  "分组信息"
// @Success      200   {object}  dto.Response{data=dto.PersonGroupResponse}
// @Router       /person-groups [post]
// @Security     BearerAuth
func (h *PersonHandler) CreateGroup(c *gin.Context) {
	var req dto.PersonGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	item, err := h.svc.CreateGroup(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}

// UpdateGroup 更新分组。
// @Summary      更新分组
// @Tags         人员分组
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "分组 ID"
// @Param        body  body  dto.PersonGroupRequest  true  "分组信息"
// @Success      200   {object}  dto.Response{data=dto.PersonGroupResponse}
// @Router       /person-groups/{id} [put]
// @Security     BearerAuth
func (h *PersonHandler) UpdateGroup(c *gin.Context) {
	var req dto.PersonGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	item, err := h.svc.UpdateGroup(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}

// DeleteGroup 删除分组。
// @Summary      删除分组
// @Tags         人员分组
// @Produce      json
// @Param        id  path  string  true  "分组 ID"
// @Success      200  {object}  dto.Response
// @Router       /person-groups/{id} [delete]
// @Security     BearerAuth
func (h *PersonHandler) DeleteGroup(c *gin.Context) {
	if err := h.svc.DeleteGroup(c.Request.Context(), c.Param("id")); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// --- 导入任务 ---

// ListImportTasks 查询导入任务。
// @Summary      导入任务列表
// @Tags         人员导入
// @Produce      json
// @Param        page       query  int  false  "页码"  default(1)
// @Param        page_size  query  int  false  "每页数量"  default(20)
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.PersonImportTaskResponse}}
// @Router       /person-import-tasks [get]
// @Security     BearerAuth
func (h *PersonHandler) ListImportTasks(c *gin.Context) {
	var req dto.PageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	items, total, err := h.svc.ListImportTasks(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// GetImportTask 获取导入任务详情。
// @Summary      导入任务详情
// @Tags         人员导入
// @Produce      json
// @Param        id  path  string  true  "任务 ID"
// @Success      200  {object}  dto.Response{data=dto.PersonImportTaskResponse}
// @Router       /person-import-tasks/{id} [get]
// @Security     BearerAuth
func (h *PersonHandler) GetImportTask(c *gin.Context) {
	item, err := h.svc.GetImportTask(c.Request.Context(), c.Param("id"))
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}

// Import 上传压缩包导入人员。
// @Summary      批量导入人员
// @Description  上传压缩包（ZIP/TAR.GZ/TAR.BZ2），图片文件名将作为人员姓名（支持"编号_姓名"格式）
// @Tags         人员导入
// @Accept       multipart/form-data
// @Produce      json
// @Param        file                  formData  file    true   "压缩包（内含人脸图片，支持ZIP/TAR.GZ/TAR.BZ2）"
// @Param        overwrite_on_duplicate formData  bool    false  "重复时覆盖"
// @Success      200   {object}  dto.Response{data=dto.PersonImportTaskResponse}
// @Router       /person-import-tasks [post]
// @Security     BearerAuth
func (h *PersonHandler) Import(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrArchiveRequired, ""))
		return
	}
	overwrite, _ := strconv.ParseBool(c.PostForm("overwrite_on_duplicate"))
	item, err := h.svc.CreateImportTask(c.Request.Context(), fileHeader, overwrite)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}

// ImportByURL 通过已上传的文件 URL 创建导入任务（分片上传后调用）。
// @Summary      通过文件URL批量导入人员
// @Description  传入分片上传完成后的文件路径，创建异步导入任务
// @Tags         人员导入
// @Accept       json
// @Produce      json
// @Param        body  body  dto.PersonImportByURLRequest  true  "文件URL和选项"
// @Success      200   {object}  dto.Response{data=dto.PersonImportTaskResponse}
// @Router       /person-import-tasks/by-url [post]
// @Security     BearerAuth
func (h *PersonHandler) ImportByURL(c *gin.Context) {
	var req dto.PersonImportByURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	item, err := h.svc.CreateImportTaskByURL(c.Request.Context(), req.FileURL, req.OverwriteOnDuplicate)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}
