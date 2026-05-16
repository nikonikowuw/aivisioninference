// Package middleware provides Gin HTTP middleware for niko-admin.
package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// Recovery returns a Gin middleware that recovers from panics and returns
// a unified error response with error code 50001. It wraps gin.Recovery()
// but formats the panic as a structured error via c.Error().
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				zap.L().Error("panic recovered",
					zap.Any("error", r),
					zap.String("stack", string(debug.Stack())),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
				)

				response.Err(c, apperrors.New(apperrors.ErrInternal, "服务器内部错误"))
				c.Abort()
			}
		}()

		c.Next()
	}
}
