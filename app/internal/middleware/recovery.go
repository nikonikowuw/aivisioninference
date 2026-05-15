// Package middleware provides Gin HTTP middleware for niko-admin.
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
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

				// Create a unified error and attach to context
				err := apperrors.New(apperrors.ErrInternal, fmt.Sprintf("服务器内部错误: %v", r))
				c.Error(err)

				// Abort with unified error format
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    err.Code,
					"message": err.Message,
				})
			}
		}()

		c.Next()
	}
}
