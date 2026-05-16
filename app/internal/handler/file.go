package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// FileHandler handles HTTP requests for file upload and management.
type FileHandler struct {
	svc *service.FileService
}

// NewFileHandler creates a new FileHandler with the given dependencies.
func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

// InitUpload initializes a chunked upload session and returns an upload_id.
//
// @Summary      初始化分片上传
// @Description  创建分片上传会话，返回 upload_id
// @Tags         文件管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.InitUploadRequest  true  "上传信息"
// @Success      200   {object}  dto.Response{data=model.FileChunk}
// @Router       /files/upload/init [post]
// @Security     BearerAuth
func (h *FileHandler) InitUpload(c *gin.Context) {
	var req dto.InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	chunk, err := h.svc.InitUpload(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, chunk)
}

// UploadChunk uploads a single chunk for a resumable upload.
//
// @Summary      上传分片
// @Description  上传单个分片到指定 upload_id
// @Tags         文件管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        upload_id  path   string  true  "上传会话 ID"
// @Param        index      formData int    true  "分片索引（从 0 开始）"
// @Param        chunk      formData file   true  "分片文件"
// @Success      200   {object}  dto.Response
// @Router       /files/upload/{upload_id}/chunk [post]
// @Security     BearerAuth
func (h *FileHandler) UploadChunk(c *gin.Context) {
	uploadID := c.Param("upload_id")

	indexStr := c.PostForm("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, "无效的分片索引"))
		return
	}

	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, "缺少分片文件"))
		return
	}
	defer file.Close()

	if err := h.svc.SaveChunk(c.Request.Context(), uploadID, index, file); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// CompleteUpload merges all chunks and creates the final file record.
//
// @Summary      完成分片上传
// @Description  合并所有分片，保存文件并创建文件记录
// @Tags         文件管理
// @Produce      json
// @Param        upload_id  path   string  true  "上传会话 ID"
// @Success      200   {object}  dto.Response{data=model.File}
// @Router       /files/upload/{upload_id}/complete [post]
// @Security     BearerAuth
func (h *FileHandler) CompleteUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	fileRecord, err := h.svc.CompleteUpload(c.Request.Context(), uploadID)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, fileRecord)
}

// UploadProgress returns which chunks have been uploaded for a given upload_id.
//
// @Summary      查询上传进度
// @Description  返回已上传的分片列表
// @Tags         文件管理
// @Produce      json
// @Param        upload_id  path   string  true  "上传会话 ID"
// @Success      200   {object}  dto.Response{data=dto.UploadProgressResponse}
// @Router       /files/upload/{upload_id}/progress [get]
// @Security     BearerAuth
func (h *FileHandler) UploadProgress(c *gin.Context) {
	uploadID := c.Param("upload_id")

	progress, err := h.svc.GetUploadProgress(c.Request.Context(), uploadID)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, progress)
}

// CheckFile checks if a file with the given MD5 already exists (for instant upload).
//
// @Summary      秒传检查
// @Description  根据 MD5 检查文件是否已存在
// @Tags         文件管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CheckFileRequest  true  "MD5 值"
// @Success      200   {object}  dto.Response{data=dto.CheckFileResponse}
// @Router       /files/upload/check [post]
// @Security     BearerAuth
func (h *FileHandler) CheckFile(c *gin.Context) {
	var req dto.CheckFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	result, err := h.svc.CheckFile(c.Request.Context(), req.MD5)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, result)
}

// List returns a paginated list of files with optional filters.
//
// @Summary      文件列表
// @Description  分页查询文件列表，支持按关键词、MIME 类型、存储类型筛选
// @Tags         文件管理
// @Produce      json
// @Param        page         query   int     false  "页码"       default(1)
// @Param        page_size    query   int     false  "每页数量"   default(20)
// @Param        keyword      query   string  false  "关键词搜索（文件名/原始名）"
// @Param        mime_type    query   string  false  "MIME 类型筛选"
// @Param        storage_type query   string  false  "存储类型筛选"
// @Param        uploader_id  query   string  false  "上传者 ID 筛选"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.File}}
// @Router       /files [get]
// @Security     BearerAuth
func (h *FileHandler) List(c *gin.Context) {
	var req dto.FileListRequest
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

// GetByID returns file information by its ID.
//
// @Summary      获取文件详情
// @Description  根据 ID 查询文件信息
// @Tags         文件管理
// @Produce      json
// @Param        id   path   string  true  "文件 ID"
// @Success      200  {object}  dto.Response{data=model.File}
// @Router       /files/{id} [get]
// @Security     BearerAuth
func (h *FileHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	file, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, file)
}

// Download serves the file with Range header support for partial downloads.
//
// @Summary      下载文件
// @Description  下载文件，支持 Range 请求（断点续传）
// @Tags         文件管理
// @Produce      octet-stream
// @Param        id   path   string  true  "文件 ID"
// @Success      200  {file}  binary
// @Router       /files/{id}/download [get]
// @Security     BearerAuth
func (h *FileHandler) Download(c *gin.Context) {
	id := c.Param("id")
	rangeHeader := c.GetHeader("Range")

	info, err := h.svc.GetDownloadInfo(c.Request.Context(), id, rangeHeader)
	if err != nil {
		attachError(c, err)
		return
	}

	if rangeHeader != "" {
		start, end, ok := service.ParseRange(rangeHeader, info.FileSize)
		if !ok {
			c.Header("Content-Range", fmt.Sprintf("bytes */%d", info.FileSize))
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		if err := service.WriteRange(c.Writer, info.FilePath, start, end, info.FileSize, info.ContentType); err != nil {
			zap.L().Error("write range failed", zap.Error(err))
			attachError(c, apperrors.New(apperrors.ErrInternal, ""))
			return
		}
		return
	}

	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, info.File.OriginalName))
	c.Header("Content-Length", strconv.FormatInt(info.FileSize, 10))
	c.Header("Accept-Ranges", "bytes")
	c.File(info.FilePath)
}

// Delete soft-deletes a file record and removes the physical file.
//
// @Summary      删除文件
// @Description  软删除文件记录并删除物理文件
// @Tags         文件管理
// @Produce      json
// @Param        id  path  string  true  "文件 ID"
// @Success      200  {object}  dto.Response
// @Router       /files/{id} [delete]
// @Security     BearerAuth
func (h *FileHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}
