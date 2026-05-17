package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/niko-admin/niko-admin/internal/middleware"
)

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setupCtx   func(c *gin.Context)
		wantID     string
		wantOK     bool
		wantErrors int
	}{
		{
			name: "valid user ID",
			setupCtx: func(c *gin.Context) {
				c.Set(middleware.ContextKeyUserID, "user-123")
			},
			wantID:     "user-123",
			wantOK:     true,
			wantErrors: 0,
		},
		{
			name:       "missing user ID",
			setupCtx:   func(c *gin.Context) {},
			wantID:     "",
			wantOK:     false,
			wantErrors: 1,
		},
		{
			name: "non-string user ID",
			setupCtx: func(c *gin.Context) {
				c.Set(middleware.ContextKeyUserID, 12345)
			},
			wantID:     "",
			wantOK:     false,
			wantErrors: 1,
		},
		{
			name: "empty string user ID",
			setupCtx: func(c *gin.Context) {
				c.Set(middleware.ContextKeyUserID, "")
			},
			wantID:     "",
			wantOK:     false,
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setupCtx(c)

			uid, ok := getUserID(c)
			assert.Equal(t, tt.wantID, uid)
			assert.Equal(t, tt.wantOK, ok)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}
