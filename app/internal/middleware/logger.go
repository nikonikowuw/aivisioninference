// Package middleware 提供 Gin HTTP 中间件，包含认证鉴权、RBAC 权限控制、审计日志、CORS、限流等功能。
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger returns a Gin middleware that logs each request using the provided
// access logger with structured fields for method, path, status, latency, and client IP.
// 它自动从 gin.Context 中提取 TraceID（由 TraceID 中间件注入）和 user_id（由 Auth 中间件注入）。
func Logger(accessLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("body_size", c.Writer.Size()),
			zap.Int("error_count", len(c.Errors)),
		}

		// 追加追踪字段：trace_id（TraceID 中间件）和 user_id（Auth 中间件）
		fields = append(fields, ginTraceFields(c)...)

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("last_error", c.Errors.Last().Error()))
		}

		switch {
		case statusCode >= 500:
			accessLogger.Error("server error", fields...)
		case statusCode >= 400:
			accessLogger.Warn("client error", fields...)
		default:
			accessLogger.Info("request", fields...)
		}
	}
}
