package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
)

// MockDeviceRepo is a mock of deviceRepo interface
type MockDeviceRepo struct {
	mock.Mock
}

func (m *MockDeviceRepo) List(ctx context.Context, req dto.DeviceListRequest) ([]model.Device, int64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]model.Device), args.Get(1).(int64), args.Error(2)
}

func (m *MockDeviceRepo) FindByID(ctx context.Context, id string) (*model.Device, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Device), args.Error(1)
}

func (m *MockDeviceRepo) FindByIDs(ctx context.Context, ids []string) ([]model.Device, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]model.Device), args.Error(1)
}

func (m *MockDeviceRepo) ExistsByName(ctx context.Context, name, excludeID string) (bool, error) {
	args := m.Called(ctx, name, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDeviceRepo) Create(ctx context.Context, item *model.Device) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockDeviceRepo) Update(ctx context.Context, item *model.Device) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockDeviceRepo) UpdateStatus(ctx context.Context, id, status, errorCode, errorMessage string) error {
	args := m.Called(ctx, id, status, errorCode, errorMessage)
	return args.Error(0)
}

func (m *MockDeviceRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDeviceRepo) BatchDelete(ctx context.Context, ids []string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockDeviceRepo) ListByGroupID(ctx context.Context, groupID string) ([]model.Device, error) {
	args := m.Called(ctx, groupID)
	return args.Get(0).([]model.Device), args.Error(1)
}

// MockCache is a mock of cache interface
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCache) Del(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// MockTaskClient is a mock of taskClient interface
type MockTaskClient struct {
	mock.Mock
}

func (m *MockTaskClient) Enqueue(ctx context.Context, taskType string, payload interface{}) error {
	args := m.Called(ctx, taskType, payload)
	return args.Error(0)
}

// MockZLMClient is a mock of zlmClient interface
type MockZLMClient struct {
	mock.Mock
}

func (m *MockZLMClient) AddStreamProxy(ctx context.Context, req zlm.AddStreamProxyRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *MockZLMClient) CloseStream(ctx context.Context, req zlm.CloseStreamRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockZLMClient) IsMediaOnline(ctx context.Context, schema, vhost, app, stream string) (bool, error) {
	args := m.Called(ctx, schema, vhost, app, stream)
	return args.Bool(0), args.Error(1)
}

func TestDeviceService_Integration_Workflow(t *testing.T) {
	mockRepo := new(MockDeviceRepo)
	mockCache := new(MockCache)
	mockTask := new(MockTaskClient)
	mockZLM := new(MockZLMClient)
	svc := NewDeviceService(mockRepo, mockCache, mockTask, mockZLM)

	ctx := context.Background()

	t.Run("Create Device with Transaction and Task Enqueue", func(t *testing.T) {
		req := dto.DeviceCreateRequest{
			DeviceName: "Test Device",
			AccessType: "rtsp",
			RtspURL:    "rtsp://example.com/stream",
		}

		mockRepo.On("ExistsByName", ctx, req.DeviceName, "").Return(false, nil)
		mockRepo.On("Create", ctx, mock.MatchedBy(func(d *model.Device) bool {
			return d.DeviceName == req.DeviceName
		})).Return(nil).Run(func(args mock.Arguments) {
			d := args.Get(1).(*model.Device)
			d.ID = "generated-id"
		})

		// 验证缓存被设置
		mockCache.On("Set", ctx, mock.AnythingOfType("string"), mock.Anything, 24*time.Hour).Return(nil)
		// 验证探测任务被排队
		mockTask.On("Enqueue", ctx, "device:detect", mock.MatchedBy(func(p interface{}) bool {
			payload := p.(map[string]string)
			return payload["id"] != ""
		})).Return(nil)

		res, err := svc.Create(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, req.DeviceName, res.DeviceName)
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
		mockTask.AssertExpectations(t)
	})

	t.Run("Update Device Status with Cache and State Sync", func(t *testing.T) {
		deviceID := "dev-123"
		status := "online"

		mockRepo.On("FindByID", ctx, deviceID).Return(&model.Device{
			BaseModel:  model.BaseModel{ID: deviceID},
			DeviceName: "Test Device",
			Status:     "offline",
		}, nil)

		mockRepo.On("Update", ctx, mock.MatchedBy(func(d *model.Device) bool {
			return d.ID == deviceID && d.Status == status
		})).Return(nil)

		// 验证状态变更时更新缓存
		mockCache.On("Set", ctx, mock.MatchedBy(func(k string) bool {
			return k == "device:status:dev-123"
		}), mock.Anything, 24*time.Hour).Return(nil)

		req := dto.DeviceUpdateRequest{
			Status: status,
		}

		res, err := svc.Update(ctx, deviceID, req)

		assert.NoError(t, err)
		assert.Equal(t, status, res.Status)
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}
