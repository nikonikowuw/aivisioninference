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

// FileHandler 处理文件上传和管理的 HTTP 请求。
type FileHandler struct {
	svc *service.FileService
}

// NewFileHandler 创建一个新的 FileHandler 实例。
func NewFileHandler(svc *service.FileService) *FileHandler {
	return &FileHandler{svc: svc}
}

// InitUpload 初始化分片上传会话，生成并返回唯一上传任务 ID（upload_id）。
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
	// 绑定并验证初始化请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	// 开启分片上传任务并记录元数据
	chunk, err := h.svc.InitUpload(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, chunk)
}

// UploadChunk 上传单个文件分片，支持断点续传。
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
	// 获取 URL 路径中的上传任务 ID
	uploadID := c.Param("upload_id")

	// 解析表单参数中传递的分片索引号
	indexStr := c.PostForm("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, "无效的分片索引"))
		return
	}

	// 提取表单中的分片文件二进制内容
	file, _, err := c.Request.FormFile("chunk")
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, "缺少分片文件"))
		return
	}
	defer file.Close()

	// 保存该分片到临时目录
	if err := h.svc.SaveChunk(c.Request.Context(), uploadID, index, file); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// CompleteUpload 合并所有已上传的分片，并在成功合并后创建最终的数据库文件记录。
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

	// 服务层执行合并，并计算文件 MD5 写入库中
	fileRecord, err := h.svc.CompleteUpload(c.Request.Context(), uploadID)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, fileRecord)
}

// UploadProgress 根据上传会话 ID 获取当前已成功上传的所有分片的索引，用于恢复上传。
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

	// 获取已完成分片索引
	progress, err := h.svc.GetUploadProgress(c.Request.Context(), uploadID)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, progress)
}

// CheckFile 根据文件的 MD5 哈希校验该文件是否已被他人上传过，实现“秒传”逻辑。
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
		attachError(c, badRequestError(c, err))
		return
	}

	// 执行秒传比对
	result, err := h.svc.CheckFile(c.Request.Context(), req.MD5)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, result)
}

// List 返回分页的文件列表，支持按关键词、MIME 类型和存储类型等过滤查询。
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

// GetByID 根据指定 ID 查询并返回某个文件的元数据属性详情。
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

// Download 处理文件下载请求，完美支持 HTTP Range 标头进行文件的分片/断点续传下载。
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

	// 获取文件物理下载路径和元数据
	info, err := h.svc.GetDownloadInfo(c.Request.Context(), id, rangeHeader)
	if err != nil {
		attachError(c, err)
		return
	}

	// 如果包含 Range 请求头，则按区间返回文件块，从而支持断点续传
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

	// 没有 Range 头，正常发送整个文件，附带附件下载文件名头
	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, info.File.OriginalName))
	c.Header("Content-Length", strconv.FormatInt(info.FileSize, 10))
	c.Header("Accept-Ranges", "bytes")
	c.File(info.FilePath)
}

// Delete 软删除文件对应的数据库记录，并在物理/云存储介质中彻底删除物理文件。
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
