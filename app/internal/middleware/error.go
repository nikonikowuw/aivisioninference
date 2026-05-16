package middleware

import (
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
		fields = append(fields, zap.Int("code", appErr.Code))

		switch {
		case appErr.Code >= 50000:
			zap.L().Error("server error", fields...)
		default:
			zap.L().Warn("client error", fields...)
		}
	} else {
		fields = append(fields, zap.Error(err))
		zap.L().Error("unknown error", fields...)
	}
}
