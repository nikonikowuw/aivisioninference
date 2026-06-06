package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// UploadAlgorithm 处理算法包的流式上传。
//
// @Summary      上传算法包
// @Description  使用 MultipartReader 流式上传算法包，防止 OOM，并自动进行安全重打包
// @Tags         algorithmpackage
// @Accept       multipart/form-data
// @Produce      json
// @Success      200   {object}  dto.Response{data=model.AlgorithmPackage}
// @Router       /algorithmpackages/upload [post]
// @Security     BearerAuth
func (h *AlgorithmPackageHandler) UploadAlgorithm(c *gin.Context) {
	// 1. 限制请求大小
	maxSize := h.svc.MaxAlgoRequestBytes()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

	// 2. 获取 multipart reader
	mr, err := c.Request.MultipartReader()
	if err != nil {
		zap.L().Warn("[AlgorithmPackage] invalid multipart form",
			zap.Error(err),
			zap.String("content_type", c.Request.Header.Get("Content-Type")),
		)
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}

	// 3. 流式读取并处理
	item, err := h.svc.UploadAndProcess(c.Request.Context(), mr)
	if err != nil {
		zap.L().Warn("[AlgorithmPackage] upload failed",
			zap.Error(err),
		)
		response.Err(c, err)
		return
	}

	response.OK(c, item)
}

// DownloadAlgorithmPackage 处理自检时的算法包下载。
//
// @Summary      下载算法包
// @Description  使用短效 Token 安全下载 algorithm package .tar 文件
// @Tags         algorithmpackage
// @Produce      octet-stream
// @Param        token  query   string  true  "下载 Token"
// @Success      200    {file}  binary
// @Router       /internal/algo/download [get]
func (h *AlgorithmPackageHandler) DownloadAlgorithmPackage(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}

	// Retrieve tar path from redis
	tarPath, err := h.svc.GetPathByToken(c.Request.Context(), token)
	if err != nil || tarPath == "" {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, ""))
		return
	}

	// Send file
	c.Header("Content-Disposition", "attachment; filename=algorithm.tar")
	c.Header("Content-Type", "application/x-tar")
	c.File(tarPath)
}
