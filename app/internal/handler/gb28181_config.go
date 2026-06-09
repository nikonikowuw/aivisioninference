package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
)

type GB28181ConfigHandler struct {
	zlmClient *zlm.Client
}

func NewGB28181ConfigHandler(zlmClient *zlm.Client) *GB28181ConfigHandler {
	return &GB28181ConfigHandler{zlmClient: zlmClient}
}

// GetConfig 获取 ZLM GB28181 配置
// @Summary      获取 ZLM GB28181 配置
// @Tags         GB28181配置
// @Produce      json
// @Success      200  {object}  dto.Response
// @Router       /system/gb28181/config [get]
// @Security     BearerAuth
func (h *GB28181ConfigHandler) GetConfig(c *gin.Context) {
	if h.zlmClient == nil {
		response.Err(c, badRequestError(c, nil))
		return
	}

	config, err := h.zlmClient.GetServerConfig(c.Request.Context())
	if err != nil {
		attachError(c, err)
		return
	}

	// Filter only sip configs
	gbConfig := make(map[string]interface{})
	if len(config) > 0 {
		for _, cfg := range config {
			for k, v := range cfg {
				// The spec states: SIP端口/域/密码/心跳/同步间隔 etc. In ZLM, they usually prefix with "sip."
				if strings.HasPrefix(k, "sip.") {
					gbConfig[k] = v
				}
			}
		}
	}

	response.OK(c, gbConfig)
}

// UpdateConfig 更新 ZLM GB28181 配置
// @Summary      更新 ZLM GB28181 配置
// @Tags         GB28181配置
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]interface{}  true  "GB28181配置"
// @Success      200  {object}  dto.Response
// @Router       /system/gb28181/config [put]
// @Security     BearerAuth
func (h *GB28181ConfigHandler) UpdateConfig(c *gin.Context) {
	if h.zlmClient == nil {
		response.Err(c, badRequestError(c, nil))
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	gbConfig := make(map[string]interface{})
	for key, value := range req {
		if !strings.HasPrefix(key, "sip.") {
			continue
		}
		gbConfig[key] = value
	}

	if len(gbConfig) == 0 {
		response.Err(c, badRequestError(c, nil))
		return
	}

	if err := h.zlmClient.SetServerConfig(c.Request.Context(), gbConfig); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, gbConfig)
}
