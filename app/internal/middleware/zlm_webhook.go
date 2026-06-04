package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZLMWebhookAuth validates that webhook requests come from ZLMediaKit.
// It checks the presence of a pre-shared secret in headers.
func ZLMWebhookAuth(expectedSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := c.GetHeader("X-ZLM-Secret")
		if secret == "" {
			secret = c.Query("secret")
		}

		if secret == "" || secret != expectedSecret {
			zap.L().Warn("ZLM webhook auth failed",
				zap.String("remote_ip", c.ClientIP()),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": -1, "msg": "unauthorized"})
			return
		}

		c.Next()
	}
}
