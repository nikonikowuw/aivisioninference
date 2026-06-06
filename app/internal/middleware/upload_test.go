package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUploadProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.Use(UploadProtection(1))

	hold := make(chan struct{})

	r.POST("/upload", func(c *gin.Context) {
		<-hold
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	var wg sync.WaitGroup
	wg.Add(1)

	var code1, code2 int

	// First request - will block on hold
	go func() {
		defer wg.Done()
		req := httptest.NewRequest(http.MethodPost, "/upload", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		code1 = w.Code
	}()

	// Wait a bit to ensure the first request enters the handler
	time.Sleep(50 * time.Millisecond)

	// Second request - should fail with 429
	req2 := httptest.NewRequest(http.MethodPost, "/upload", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	code2 = w2.Code

	// Release first request
	close(hold)
	wg.Wait()

	require.Equal(t, http.StatusOK, code1)
	require.Equal(t, http.StatusTooManyRequests, code2)
}
