package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// CORS returns a Gin middleware that configures Cross-Origin Resource Sharing
// with the provided allowed origins. If allowOrigins is empty, it defaults
// to allowing all origins ("*").
//
// Unlike gin-contrib/cors, this implementation returns a proper JSON error
// response (instead of an empty body) when a request is rejected due to
// a disallowed origin.
func CORS(allowOrigins []string) gin.HandlerFunc {
	allowAll := false
	originSet := make(map[string]bool, len(allowOrigins))
	for _, o := range allowOrigins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 无 Origin 头（非浏览器请求），放行
		if origin == "" {
			c.Next()
			return
		}

		// 检查 Origin 是否允许
		if !allowAll && !originSet[origin] {
			zap.L().Warn("cors origin rejected",
				zap.String("origin", origin),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
				zap.String("user_agent", c.Request.UserAgent()),
				zap.String("request_id", c.GetHeader("X-Request-ID")),
			)
			c.Abort()
			response.Err(c, apperrors.New(apperrors.ErrOriginNotAllowed, ""))
			return
		}

		// Vary 头告知缓存响应因 Origin 而异
		c.Header("Vary", "Origin")

		// 设置 CORS 响应头
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Accept,Authorization,X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length,X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "43200") // 12 hours

		// 预检请求直接返回 204
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
