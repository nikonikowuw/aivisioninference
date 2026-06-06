// Package handler 提供 HTTP 请求处理层（Controller）。
package handler

import (
	"io"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// LicenseHandler 处理授权管理相关的 HTTP 请求。
type LicenseHandler struct {
	svc *service.LicenseService
}

// NewLicenseHandler 创建一个新的 LicenseHandler 实例。
func NewLicenseHandler(svc *service.LicenseService) *LicenseHandler {
	return &LicenseHandler{svc: svc}
}

// GetFingerprint 返回本机设备指纹，供前端一键复制。
//
// @Summary      获取设备指纹
// @Description  提取本机硬件指纹用于授权绑定
// @Tags         授权管理
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.FingerprintResponse}
// @Router       /license/fingerprint [get]
// @Security     BearerAuth
func (h *LicenseHandler) GetFingerprint(c *gin.Context) {
	result, err := h.svc.GetDeviceFingerprint(c.Request.Context())
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, result)
}

// Upload 解析上传的 .lic 授权文件并执行离线验签。
//
// @Summary      上传授权文件
// @Description  上传 .lic JWT 授权文件，本地离线验签并校验设备指纹
// @Tags         授权管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "授权文件 (.lic)"
// @Success      200   {object}  dto.Response{data=dto.LicenseInfoResponse}
// @Router       /license/upload [post]
// @Security     BearerAuth
func (h *LicenseHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		attachError(c, errors.New(errors.ErrBadRequest, "请上传授权文件"))
		return
	}
	defer file.Close()

	// 校验文件大小（最大 1MB）
	if header.Size > 1024*1024 {
		attachError(c, errors.New(errors.ErrFileTooLarge, "授权文件大小不能超过 1MB"))
		return
	}

	rawBytes, err := io.ReadAll(file)
	if err != nil {
		attachError(c, errors.New(errors.ErrInternal, "读取授权文件失败"))
		return
	}
	rawJWT := string(rawBytes)

	result, uploadErr := h.svc.Upload(c.Request.Context(), rawJWT)
	if uploadErr != nil {
		attachError(c, uploadErr)
		return
	}

	response.OK(c, result)
}

// GetActive 获取当前生效的授权信息。
//
// @Summary      获取当前授权信息
// @Description  查询当前设备生效的授权状态和详情
// @Tags         授权管理
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.LicenseInfoResponse}
// @Router       /license/active [get]
// @Security     BearerAuth
func (h *LicenseHandler) GetActive(c *gin.Context) {
	result, err := h.svc.GetActive(c.Request.Context())
	if err != nil {
		// 未上传或未找到有效授权是正常空状态，避免前端启动时出现 HTTP 404 资源错误。
		if appErr, ok := err.(*errors.AppError); ok && appErr.Code == errors.ErrNotFound {
			response.OK(c, nil)
			return
		}
		attachError(c, err)
		return
	}
	response.OK(c, result)
}

// List 返回分页的授权列表。
//
// @Summary      授权列表
// @Description  分页查询授权列表
// @Tags         授权管理
// @Produce      json
// @Param        page       query   int     false  "页码"       default(1)
// @Param        page_size  query   int     false  "每页数量"   default(20)
// @Param        keyword    query   string  false  "关键词搜索"
// @Param        status     query   string  false  "状态筛选"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.LicenseInfoResponse}}
// @Router       /license [get]
// @Security     BearerAuth
func (h *LicenseHandler) List(c *gin.Context) {
	var req dto.LicenseListRequest
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

// CheckAuth 校验指定算法的授权状态（供内部其他 Service 调用时的 HTTP 入口）。
//
// @Summary      校验算法授权
// @Description  检查指定算法是否被授权且授权有效
// @Tags         授权管理
// @Produce      json
// @Param        algorithm  query   string  true  "算法名称"
// @Success      200        {object}  dto.Response
// @Router       /license/check [get]
// @Security     BearerAuth
func (h *LicenseHandler) CheckAuth(c *gin.Context) {
	algoName := c.Query("algorithm")
	if algoName == "" {
		attachError(c, errors.New(errors.ErrBadRequest, "请指定算法名称"))
		return
	}

	if err := h.svc.CheckAlgorithmAuth(c.Request.Context(), algoName); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, gin.H{"authorized": true, "algorithm": algoName})
}
