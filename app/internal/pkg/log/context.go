// Package log 提供集中式日志初始化与全链路追踪上下文集成。
package log

import (
	"context"

	"go.uber.org/zap"
)

// LogContext 携带请求链路的追踪上下文信息，通过 context.Context 传递。
type LogContext struct {
	TraceID  string
	UserID   string
	TenantID string
	SpanID   string
}

// logContextKey 是 context.Value 用的私有 key，避免与第三方库冲突。
type logContextKey struct{}

// WithLogContext 将 LogContext 注入到 context.Context 中。
func WithLogContext(ctx context.Context, lc LogContext) context.Context {
	return context.WithValue(ctx, logContextKey{}, lc)
}

// GetLogContext 从 context.Context 中提取 LogContext。
// 第二个返回值表示是否存在有效的 LogContext。
func GetLogContext(ctx context.Context) (LogContext, bool) {
	lc, ok := ctx.Value(logContextKey{}).(LogContext)
	return lc, ok
}

// Ctx 返回一个带有全链路追踪字段的 zap.Logger，使用 zap.L() 作为基底。
// 当 context 中不存在 LogContext 时，直接返回 zap.L() 而不追加字段。
//
// 使用示例：
//
//	log.Ctx(ctx).Info("device created")
//	// 输出: {"trace_id":"xxx","user_id":"yyy","msg":"device created"}
func Ctx(ctx context.Context) *zap.Logger {
	return addContextFields(zap.L(), ctx)
}

// addContextFields 从 context 提取 LogContext 字段并追加到 logger。
func addContextFields(base *zap.Logger, ctx context.Context) *zap.Logger {
	lc, ok := GetLogContext(ctx)
	if !ok {
		return base
	}
	fields := make([]zap.Field, 0, 4)
	if lc.TraceID != "" {
		fields = append(fields, zap.String("trace_id", lc.TraceID))
	}
	if lc.UserID != "" {
		fields = append(fields, zap.String("user_id", lc.UserID))
	}
	if lc.SpanID != "" {
		fields = append(fields, zap.String("span_id", lc.SpanID))
	}
	if lc.TenantID != "" {
		fields = append(fields, zap.String("tenant_id", lc.TenantID))
	}
	if len(fields) > 0 {
		return base.With(fields...)
	}
	return base
}
