package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// attachError 将错误挂载到 Gin Context，供全局错误中间件统一响应。
func attachError(c *gin.Context, err error) {
	if attachErr := c.Error(err); attachErr != nil {
		zap.L().Warn("attach gin error failed", zap.Error(attachErr))
	}
}
