package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
	"github.com/niko-admin/niko-admin/internal/service"
)

type GB28181ConfigHandler struct {
	zlmClient         *zlm.Client
	platformConfigSvc *service.GB28181PlatformConfigService
}

func NewGB28181ConfigHandler(zlmClient *zlm.Client, platformConfigSvc *service.GB28181PlatformConfigService) *GB28181ConfigHandler {
	return &GB28181ConfigHandler{
		zlmClient:         zlmClient,
		platformConfigSvc: platformConfigSvc,
	}
}

// GetConfig 获取 GB28181 配置
// @Summary      获取 GB28181 配置
// @Tags         GB28181配置
// @Produce      json
// @Success      200  {object}  dto.Response
// @Router       /system/gb28181/config [get]
// @Security     BearerAuth
func (h *GB28181ConfigHandler) GetConfig(c *gin.Context) {
	ctx := c.Request.Context()
	cfg, err := h.platformConfigSvc.GetConfig(ctx)
	if err != nil {
		attachError(c, err)
		return
	}

	// Fetch ZLM config for fallback/backward-compatibility keys if needed
	zlmConfig := make(map[string]interface{})
	if h.zlmClient != nil {
		if configList, err := h.zlmClient.GetServerConfig(ctx); err == nil && len(configList) > 0 {
			for _, cfgMap := range configList {
				for k, v := range cfgMap {
					if strings.HasPrefix(k, "sip.") {
						zlmConfig[k] = v
					}
				}
			}
		}
	}

	// Construct flat map
	gbConfig := make(map[string]interface{})
	for k, v := range zlmConfig {
		gbConfig[k] = v
	}

	// Override with our platform configs
	gbConfig["sip.enabled"] = cfg.Enabled
	gbConfig["sip.id"] = cfg.SipID
	gbConfig["sip.domain"] = cfg.SipDomain
	gbConfig["sip.realm"] = cfg.SipRealm
	gbConfig["sip.password"] = "******" // Mask password
	gbConfig["sip.listen_ip"] = cfg.ListenIP
	gbConfig["sip.port"] = cfg.ListenPort
	gbConfig["sip.transport"] = cfg.Transport
	gbConfig["sip.advertised_ip"] = cfg.AdvertisedIP
	gbConfig["sip.rtp_ip"] = cfg.RtpIP
	gbConfig["sip.heartbeat_timeout"] = cfg.HeartbeatTimeout
	gbConfig["sip.catalog_interval"] = cfg.CatalogInterval

	response.OK(c, gbConfig)
}

// UpdateConfig 更新 GB28181 配置
// @Summary      更新 GB28181 配置
// @Tags         GB28181配置
// @Accept       json
// @Produce      json
// @Param        body  body  map[string]interface{}  true  "GB28181配置"
// @Success      200  {object}  dto.Response
// @Router       /system/gb28181/config [put]
// @Security     BearerAuth
func (h *GB28181ConfigHandler) UpdateConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	ctx := c.Request.Context()
	currentCfg, err := h.platformConfigSvc.GetConfig(ctx)
	if err != nil {
		attachError(c, err)
		return
	}

	// Prepare update request DTO
	updateReq := dto.GB28181PlatformConfigRequest{
		Enabled:          currentCfg.Enabled,
		SipID:            currentCfg.SipID,
		SipDomain:         currentCfg.SipDomain,
		SipRealm:          currentCfg.SipRealm,
		SipPassword:       "", // keep unchanged by default
		ListenIP:         currentCfg.ListenIP,
		ListenPort:       currentCfg.ListenPort,
		Transport:        currentCfg.Transport,
		AdvertisedIP:     currentCfg.AdvertisedIP,
		RtpIP:            currentCfg.RtpIP,
		HeartbeatTimeout: currentCfg.HeartbeatTimeout,
		CatalogInterval:  currentCfg.CatalogInterval,
	}

	// Read fields from flat map request
	if val, ok := req["sip.enabled"]; ok {
		if b, ok := val.(bool); ok {
			updateReq.Enabled = b
		} else if s, ok := val.(string); ok {
			updateReq.Enabled = s == "true"
		}
	}
	if val, ok := req["sip.id"]; ok {
		if s, ok := val.(string); ok {
			updateReq.SipID = s
		}
	}
	if val, ok := req["sip.domain"]; ok {
		if s, ok := val.(string); ok {
			updateReq.SipDomain = s
		}
	}
	if val, ok := req["sip.realm"]; ok {
		if s, ok := val.(string); ok {
			updateReq.SipRealm = s
		}
	}
	if val, ok := req["sip.password"]; ok {
		if s, ok := val.(string); ok && s != "" && s != "******" {
			updateReq.SipPassword = s
		}
	}
	if val, ok := req["sip.listen_ip"]; ok {
		if s, ok := val.(string); ok {
			updateReq.ListenIP = s
		}
	}
	if val, ok := req["sip.port"]; ok {
		if f, ok := val.(float64); ok {
			updateReq.ListenPort = int(f)
		} else if s, ok := val.(string); ok {
			if p, err := strconv.Atoi(s); err == nil {
				updateReq.ListenPort = p
			}
		}
	}
	if val, ok := req["sip.transport"]; ok {
		if s, ok := val.(string); ok {
			updateReq.Transport = s
		}
	}
	if val, ok := req["sip.advertised_ip"]; ok {
		if s, ok := val.(string); ok {
			updateReq.AdvertisedIP = s
		}
	}
	if val, ok := req["sip.rtp_ip"]; ok {
		if s, ok := val.(string); ok {
			updateReq.RtpIP = s
		}
	}
	if val, ok := req["sip.heartbeat_timeout"]; ok {
		if f, ok := val.(float64); ok {
			updateReq.HeartbeatTimeout = int(f)
		} else if s, ok := val.(string); ok {
			if t, err := strconv.Atoi(s); err == nil {
				updateReq.HeartbeatTimeout = t
			}
		}
	}
	if val, ok := req["sip.catalog_interval"]; ok {
		if f, ok := val.(float64); ok {
			updateReq.CatalogInterval = int(f)
		} else if s, ok := val.(string); ok {
			if t, err := strconv.Atoi(s); err == nil {
				updateReq.CatalogInterval = t
			}
		}
	}

	updatedCfg, err := h.platformConfigSvc.UpdateConfig(ctx, updateReq)
	if err != nil {
		attachError(c, err)
		return
	}

	// Update ZLM config if needed (best-effort, non-fatal)
	if h.zlmClient != nil {
		zlmUpdates := make(map[string]interface{})
		// For backward compatibility, if zlm is active we write some fields back to ZLM
		zlmUpdates["sip.port"] = strconv.Itoa(updatedCfg.ListenPort)
		zlmUpdates["sip.id"] = updatedCfg.SipID
		zlmUpdates["sip.domain"] = updatedCfg.SipDomain
		if updateReq.SipPassword != "" {
			zlmUpdates["sip.password"] = updatedCfg.SipPassword
		}
		// Best-effort: ZLM may not be running
		_ = h.zlmClient.SetServerConfig(ctx, zlmUpdates)
	}

	// Return typed response DTO (never expose password)
	resp := dto.GB28181PlatformConfigResponse{
		Enabled:          updatedCfg.Enabled,
		SipID:            updatedCfg.SipID,
		SipDomain:        updatedCfg.SipDomain,
		SipRealm:         updatedCfg.SipRealm,
		ListenIP:         updatedCfg.ListenIP,
		ListenPort:       updatedCfg.ListenPort,
		Transport:        updatedCfg.Transport,
		AdvertisedIP:     updatedCfg.AdvertisedIP,
		RtpIP:            updatedCfg.RtpIP,
		HeartbeatTimeout: updatedCfg.HeartbeatTimeout,
		CatalogInterval:  updatedCfg.CatalogInterval,
		CreatedAt:        updatedCfg.CreatedAt,
		UpdatedAt:        updatedCfg.UpdatedAt,
	}

	response.OK(c, resp)
}
