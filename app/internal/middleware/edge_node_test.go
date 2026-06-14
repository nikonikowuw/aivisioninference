package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/pkg/jwt"
)

func TestEdgeNodeMiddleware_AuthNode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "my-very-secure-jwt-secret-at-least-32-chars"
	m := jwt.NewManager(secret, "niko-admin", "niko-admin", 3600, 86400, nil)
	mw := NewEdgeNodeMiddleware(m)

	token, err := m.GenerateNodeToken("node-123")
	require.NoError(t, err)

	tests := []struct {
		name           string
		setupRequest   func(req *http.Request)
		pathID         string
		expectedStatus int
		expectedNodeID string
	}{
		{
			name: "Valid Token, Path Matches",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+token)
			},
			pathID:         "node-123",
			expectedStatus: http.StatusOK,
			expectedNodeID: "node-123",
		},
		{
			name: "Valid Token, No Path Param",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+token)
			},
			pathID:         "",
			expectedStatus: http.StatusOK,
			expectedNodeID: "node-123",
		},
		{
			name: "Valid Token, Path Mismatch",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+token)
			},
			pathID:         "node-456",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Missing Token",
			setupRequest: func(req *http.Request) {
			},
			pathID:         "node-123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Malformed Authorization Header",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "BearerMalformed "+token)
			},
			pathID:         "node-123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid Token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid-token-string")
			},
			pathID:         "node-123",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ErrorHandler())

			router.GET("/edge-nodes/:id/heartbeat", mw.AuthNode(), func(c *gin.Context) {
				nodeID, exists := c.Get("node_id")
				if exists {
					c.JSON(http.StatusOK, gin.H{"node_id": nodeID})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "node_id not set"})
				}
			})

			router.GET("/heartbeat", mw.AuthNode(), func(c *gin.Context) {
				nodeID, exists := c.Get("node_id")
				if exists {
					c.JSON(http.StatusOK, gin.H{"node_id": nodeID})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "node_id not set"})
				}
			})

			var path string
			if tt.pathID != "" {
				path = "/edge-nodes/" + tt.pathID + "/heartbeat"
			} else {
				path = "/heartbeat"
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, path, nil)
			tt.setupRequest(req)

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, w.Body.String(), tt.expectedNodeID)
			}
		})
	}
}
