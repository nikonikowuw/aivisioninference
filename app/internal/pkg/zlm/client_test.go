package zlm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestClient_AddStreamProxy(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/index/api/addStreamProxy", r.URL.Path)
		assert.Equal(t, "test_secret", r.URL.Query().Get("secret"))

		resp := AddStreamProxyResponse{
			ZLMRsp: ZLMRsp{Code: 0, Msg: "success"},
		}
		resp.Data.Key = "test_key"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test_secret", logger)
	key, err := client.AddStreamProxy(context.Background(), AddStreamProxyRequest{
		App:    "live",
		Stream: "test",
		URL:    "rtsp://example.com/stream",
	})

	assert.NoError(t, err)
	assert.Equal(t, "test_key", key)
}

func TestClient_ZLMError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ZLMRsp{Code: -1, Msg: "failed"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", logger)
	_, err := client.AddStreamProxy(context.Background(), AddStreamProxyRequest{})

	assert.Error(t, err)
	zlmErr, ok := err.(*ZLMError)
	assert.True(t, ok)
	assert.Equal(t, -1, zlmErr.Code)
}
