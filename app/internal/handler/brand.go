package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// BrandHandler 处理系统品牌配置的 HTTP 请求。
type BrandHandler struct {
	svc *service.BrandService
}

// NewBrandHandler 创建一个新的 BrandHandler 实例。
func NewBrandHandler(svc *service.BrandService) *BrandHandler {
	return &BrandHandler{svc: svc}
}

// GetConfig 获取系统品牌配置。
//
// @Summary      获取品牌配置
// @Tags         系统配置
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.BrandConfigResponse}
// @Router       /system/brand-config [get]
func (h *BrandHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig(c.Request.Context())
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, cfg)
}

// SaveConfig 保存系统品牌配置。
//
// @Summary      保存品牌配置
// @Tags         系统配置
// @Accept       json
// @Produce      json
// @Param        body  body  dto.BrandConfigRequest  true  "品牌配置"
// @Success      200   {object}  dto.Response{data=dto.BrandConfigResponse}
// @Router       /system/brand-config [put]
// @Security     BearerAuth
func (h *BrandHandler) SaveConfig(c *gin.Context) {
	var req dto.BrandConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	cfg, err := h.svc.SaveConfig(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, cfg)
}

// UploadLogo 上传系统 Logo 并更新品牌配置。
//
// @Summary      上传品牌 Logo
// @Tags         系统配置
// @Accept       multipart/form-data
// @Produce      json
// @Param        logo  formData  file  true  "品牌 Logo"
// @Success      200   {object}  dto.Response{data=dto.BrandLogoUploadResponse}
// @Router       /system/brand-config/logo [post]
// @Security     BearerAuth
func (h *BrandHandler) UploadLogo(c *gin.Context) {
	fileHeader, err := c.FormFile("logo")
	if err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	logoURL, err := h.svc.UploadLogo(c.Request.Context(), fileHeader)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, dto.BrandLogoUploadResponse{LogoURL: logoURL})
}
