// Package middleware 提供 Gin HTTP 中间件，包含认证鉴权、RBAC 权限控制、审计日志、CORS、限流等功能。
package middleware

import (
	"github.com/gin-gonic/gin"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// UploadProtection returns a Gin middleware that implements concurrency limiting using a semaphore.
// When the number of active requests exceeds maxConcurrency, it responds with HTTP 429.
func UploadProtection(maxConcurrency int) gin.HandlerFunc {
	if maxConcurrency <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	sem := make(chan struct{}, maxConcurrency)

	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			c.Next()
		default:
			// 直接写入响应而非仅 c.Error()，确保客户端收到 429 响应体
			response.Err(c, apperrors.New(apperrors.ErrTooManyRequests, ""))
			c.Abort()
		}
	}
}
