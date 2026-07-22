package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRateLimitNilRedisAllowsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(nil, 1))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestIsRateLimitExemptRoute(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "zlm publish callback", path: "/zlm/callback/on_publish", want: true},
		{name: "zlm play callback", path: "/zlm/callback/on_play", want: true},
		{name: "edge node heartbeat", path: "/api/v1/edge-nodes/:id/heartbeat", want: true},
		{name: "edge node list", path: "/api/v1/edge-nodes", want: false},
		{name: "similar callback path", path: "/api/v1/zlm/callback/on_publish", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isRateLimitExemptRoute(tt.path))
		})
	}
}
