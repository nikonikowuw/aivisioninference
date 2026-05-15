package middleware

import (
	"context"
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
		clientIP := c.ClientIP()
		key := fmt.Sprintf("%s%s", rateLimitPrefix, clientIP)

		ctx := context.Background()

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
			}

			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    10042,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}

		c.Next()
	}
}
