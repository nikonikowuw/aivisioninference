package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger returns a Gin middleware that logs each request using zap.L()
// structured logging with method, path, status, latency, and client IP.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		errors := c.Errors.ByType(gin.ErrorTypePrivate).String()

		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("body_size", c.Writer.Size()),
		}

		if errors != "" {
			fields = append(fields, zap.String("errors", errors))
		}

		switch {
		case statusCode >= 500:
			zap.L().Error("server error", fields...)
		case statusCode >= 400:
			zap.L().Warn("client error", fields...)
		default:
			zap.L().Info("request", fields...)
		}
	}
}
