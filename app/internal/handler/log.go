// Package handler 提供 HTTP 请求处理层（Controller），负责参数绑定、校验和响应返回。
package handler

import (
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// SetLogLevelRequest 是设置日志级别的请求体。
// 级别名与 zap.UnmarshalText 接受的规范名称一致（不含 "warning"）。
type SetLogLevelRequest struct {
	Level string `json:"level" binding:"required,oneof=debug info warn error dpanic panic fatal"`
}

// LogHandler 处理日志级别动态调整的 HTTP 请求。
type LogHandler struct {
	level *zap.AtomicLevel
}

// NewLogHandler 创建一个新的 LogHandler 实例。
// level 是日志级别的指针，所有写入 core 共享此实例，修改即时生效。
func NewLogHandler(level *zap.AtomicLevel) *LogHandler {
	return &LogHandler{level: level}
}

// SetLevel 动态设置应用日志级别。
//
// @Summary      动态调整日志级别
// @Description  运行时修改日志级别，无需重启服务。支持: debug, info, warn/warning, error, dpanic, panic, fatal。
// @Tags         系统配置
// @Accept       json
// @Produce      json
// @Param        body  body      SetLogLevelRequest  true  "日志级别"
// @Success      200   {object}  dto.Response{data=string}
// @Failure      400   {object}  dto.Response
// @Router       /api/v1/system/log/level [put]
func (h *LogHandler) SetLevel(c *gin.Context) {
	var req SetLogLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	if err := h.level.UnmarshalText([]byte(req.Level)); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, "无效的日志级别: "+err.Error()))
		return
	}

	response.OK(c, "ok")
}

// GetLevel 获取当前日志级别。
//
// @Summary      获取当前日志级别
// @Tags         系统配置
// @Produce      json
// @Success      200  {object}  dto.Response{data=string}
// @Router       /api/v1/system/log/level [get]
func (h *LogHandler) GetLevel(c *gin.Context) {
	response.OK(c, h.level.String())
}
