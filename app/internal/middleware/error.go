// Package middleware 提供 Gin HTTP 中间件，包含认证鉴权、RBAC 权限控制、审计日志、CORS、限流等功能。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// ErrorHandler returns a Gin middleware that catches errors attached to the
// context via c.Error() during handler processing, logs them at the appropriate
// severity level, and sends a unified JSON response.
//
// It must be placed after all error-producing middleware and handlers in the
// chain — typically the last middleware before the routes.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		for _, e := range c.Errors {
			logAppError(c, e.Err)
		}

		if !c.Writer.Written() {
			lastErr := c.Errors.Last().Err
			if appErr, ok := lastErr.(*apperrors.AppError); ok {
				response.Err(c, appErr)
			} else {
				response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
			}
		}
	}
}

func logAppError(c *gin.Context, err error) {
	fields := []zap.Field{
		zap.String("path", c.Request.URL.Path),
	}

	if userID, exists := c.Get(ContextKeyUserID); exists {
		if uid, ok := userID.(string); ok {
			fields = append(fields, zap.String("user_id", uid))
		}
	}

	if appErr, ok := err.(*apperrors.AppError); ok {
		fields = append(fields, zap.String("code", appErr.Code))

		switch {
		case strings.HasPrefix(appErr.Code, "INTERNAL"):
			zap.L().Error("server error", fields...)
		default:
			zap.L().Warn("client error", fields...)
		}
	} else {
		fields = append(fields, zap.Error(err))
		zap.L().Error("unknown error", fields...)
	}
}
