package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/ws"
)

type mockGB28181Service struct {
	listCalled         bool
	queryCatalogCalled bool
	lastCatalogCode    string
}

func (m *mockGB28181Service) ListGB28181Devices(ctx context.Context, req dto.GB28181DeviceListRequest) ([]model.GB28181Device, int64, error) {
	m.listCalled = true
	return []model.GB28181Device{{
		BaseModel:  model.BaseModel{ID: "gb-1"},
		DeviceCode: "34020000001320000001",
		Status:     model.GB28181StatusOnline,
	}}, 1, nil
}

func (m *mockGB28181Service) GetGB28181DeviceByID(ctx context.Context, id string) (*model.GB28181Device, error) {
	return &model.GB28181Device{
		BaseModel:  model.BaseModel{ID: id},
		DeviceCode: "34020000001320000001",
		Status:     model.GB28181StatusOnline,
	}, nil
}

func (m *mockGB28181Service) UpdateGB28181Device(ctx context.Context, id string, req dto.GB28181DeviceUpdateRequest) (*model.GB28181Device, error) {
	return &model.GB28181Device{BaseModel: model.BaseModel{ID: id}, DeviceCode: "34020000001320000001"}, nil
}

func (m *mockGB28181Service) DeleteGB28181Device(ctx context.Context, id string) error { return nil }

func (m *mockGB28181Service) QueryCatalog(ctx context.Context, deviceCode string) error {
	m.queryCatalogCalled = true
	m.lastCatalogCode = deviceCode
	return nil
}

func (m *mockGB28181Service) GetGB28181DeviceChannels(ctx context.Context, id string) ([]model.Device, error) {
	return []model.Device{{BaseModel: model.BaseModel{ID: "dev-1"}, DeviceName: "通道1", AccessType: model.DeviceAccessTypeGB28181}}, nil
}

func setupGB28181TestRouter(svc *mockGB28181Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewGB28181Handler(svc, cache.NewMemoryCache(0), ws.NewHub())
	r.GET("/gb28181/devices", h.List)
	r.POST("/gb28181/devices/:id/catalog", h.TriggerCatalog)
	return r
}

func TestGB28181HandlerList(t *testing.T) {
	svc := &mockGB28181Service{}
	r := setupGB28181TestRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/gb28181/devices?page=1&page_size=20&status=online", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.listCalled)
	assert.Contains(t, w.Body.String(), "34020000001320000001")
}

func TestGB28181HandlerTriggerCatalog(t *testing.T) {
	svc := &mockGB28181Service{}
	r := setupGB28181TestRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/gb28181/devices/gb-1/catalog", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, svc.queryCatalogCalled)
	assert.Equal(t, "34020000001320000001", svc.lastCatalogCode)

	var resp struct {
		Code int `json:"code"`
		Data dto.CatalogTaskResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data.TaskID)
}
