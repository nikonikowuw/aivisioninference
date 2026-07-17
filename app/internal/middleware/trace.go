// Package middleware 提供 Gin HTTP 中间件，包含认证鉴权、RBAC 权限控制、审计日志、CORS、限流等功能。
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	applog "github.com/niko-admin/niko-admin/internal/pkg/log"
)

// ContextKeyTraceID 是 Trace ID 在 gin.Context 中的存储 key。
const ContextKeyTraceID = "trace_id"

// ginTraceFields 从请求上下文中提取 trace_id（LogContext）和 user_id（gin.Context），
// 返回对应的 zap.Field 切片，供 Logger 和 ErrorHandler 中间件使用。
func ginTraceFields(c *gin.Context) []zap.Field {
	var fields []zap.Field

	if lc, ok := applog.GetLogContext(c.Request.Context()); ok && lc.TraceID != "" {
		fields = append(fields, zap.String("trace_id", lc.TraceID))
	}

	if userID, exists := c.Get(ContextKeyUserID); exists {
		if uid, ok := userID.(string); ok && uid != "" {
			fields = append(fields, zap.String("user_id", uid))
		}
	}

	return fields
}

// TraceID 返回一个 Gin 中间件，为每个请求生成或透传唯一 Trace ID，
// 并将 LogContext 注入到请求的 context.Context 中，供后续 handler 和日志使用。
//
// 优先级顺序（从高到低）:
//  1. X-Trace-ID 请求头
//  2. X-Request-ID 请求头
//  3. 自动生成 UUID v4
//
// 生成的 Trace ID 会写入响应头 X-Request-ID。
//
// 使用位置：必须在 Logger 中间件之前注册，确保日志能够读取到 Trace ID。
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头读取或生成 Trace ID
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = c.GetHeader("X-Request-ID")
		}
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 写入响应头，方便客户端追踪
		c.Header("X-Request-ID", traceID)

		// Auth 中间件在全局 TraceID 之后执行，此时 user_id 通常尚未设置
		lc := applog.LogContext{
			TraceID: traceID,
		}
		if userID, exists := c.Get(ContextKeyUserID); exists {
			if uid, ok := userID.(string); ok && uid != "" {
				lc.UserID = uid
			}
		}

		// 将 LogContext 注入到请求的 context.Context
		ctx := c.Request.Context()
		ctx = applog.WithLogContext(ctx, lc)
		c.Request = c.Request.WithContext(ctx)

		// 同时将 Trace ID 存入 gin.Context，方便其他中间件和 handler 快速获取
		c.Set(ContextKeyTraceID, traceID)

		c.Next()
	}
}
