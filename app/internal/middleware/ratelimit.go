// Package middleware 提供 Gin HTTP 中间件，包含认证鉴权、RBAC 权限控制、审计日志、CORS、限流等功能。
package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	rateLimitPrefix = "rate_limit:"
)

// RateLimit returns a Gin middleware that implements Redis-based sliding window
// rate limiting. It uses INCR + EXPIRE for a fixed-window approach per client IP.
// Returns HTTP 429 when the limit is exceeded.
func RateLimit(rdb *redis.Client, requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Redis 未启用或限流值无效时降级为 no-op，避免启动配置关闭 Redis 后请求 panic。
		if rdb == nil || requestsPerMinute <= 0 {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		key := fmt.Sprintf("%s%s", rateLimitPrefix, clientIP)

		ctx := c.Request.Context()

		// INCR creates the key if it doesn't exist (TTL = 0 = no expiry yet)
		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			zap.L().Error("rate limit redis error",
				zap.String("client_ip", clientIP),
				zap.Error(err),
			)
			// On Redis failure, allow the request through (fail open)
			c.Next()
			return
		}

		// Set TTL on first request (when count == 1)
		if count == 1 {
			if err := rdb.Expire(ctx, key, time.Minute).Err(); err != nil {
				zap.L().Error("rate limit expire error",
					zap.String("client_ip", clientIP),
					zap.Error(err),
				)
			}
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, int64(requestsPerMinute)-count)))

		if count > int64(requestsPerMinute) {
			// Get remaining TTL for Retry-After header
			ttl, err := rdb.TTL(ctx, key).Result()
			if err != nil {
				ttl = time.Minute
			} else if ttl == -1 {
				// Fail-safe: if the key has no TTL (e.g. Expire failed on count=1),
				// it will block forever. Set TTL now.
				rdb.Expire(ctx, key, time.Minute)
				ttl = time.Minute
			}

			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": 10042,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}

		c.Next()
	}
}
