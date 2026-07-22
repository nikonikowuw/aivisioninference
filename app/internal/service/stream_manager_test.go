package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
)

// mockDeviceRepo 用于测试的 deviceRepo 模拟实现
type mockDeviceRepo struct {
	devices map[string]*model.Device
	mu      sync.Mutex
}

func newMockDeviceRepo() *mockDeviceRepo {
	return &mockDeviceRepo{devices: make(map[string]*model.Device)}
}

func (m *mockDeviceRepo) List(ctx context.Context, req dto.DeviceListRequest) ([]model.Device, int64, error) {
	return nil, 0, nil
}

func (m *mockDeviceRepo) FindByID(ctx context.Context, id string) (*model.Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dev := m.devices[id]
	return dev, nil
}

func (m *mockDeviceRepo) FindByIDs(ctx context.Context, ids []string) ([]model.Device, error) {
	return nil, nil
}

func (m *mockDeviceRepo) ExistsByName(ctx context.Context, name, excludeID string) (bool, error) {
	return false, nil
}

func (m *mockDeviceRepo) Create(ctx context.Context, item *model.Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[item.ID] = item
	return nil
}

func (m *mockDeviceRepo) Update(ctx context.Context, item *model.Device) error { return nil }

func (m *mockDeviceRepo) UpdateStatus(ctx context.Context, id, status, errorCode, errorMessage string) error {
	return nil
}

func (m *mockDeviceRepo) Delete(ctx context.Context, id string) error { return nil }

func (m *mockDeviceRepo) BatchDelete(ctx context.Context, ids []string) error { return nil }

func (m *mockDeviceRepo) ListByGroupID(ctx context.Context, groupID string) ([]model.Device, error) {
	return nil, nil
}

func (m *mockDeviceRepo) ReplaceGroups(ctx context.Context, deviceID string, groupIDs []string) error {
	return nil
}

func (m *mockDeviceRepo) FindByExternalKey(ctx context.Context, key string) (*model.Device, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, dev := range m.devices {
		if dev.ExternalKey != nil && *dev.ExternalKey == key {
			return dev, nil
		}
	}
	return nil, nil
}

func (m *mockDeviceRepo) ListEnabled(ctx context.Context) ([]model.Device, error) {
	return nil, nil
}

// mockStreamRepo 用于测试的 streamRepo 模拟实现
type mockStreamRepo struct {
	streams          map[string]*model.MediaStream
	mu               sync.Mutex
	lastFindCtxErr   error
	lastUpdateCtxErr error
	updateCh         chan struct{}
}

func newMockStreamRepo() *mockStreamRepo {
	return &mockStreamRepo{streams: make(map[string]*model.MediaStream)}
}

func (m *mockStreamRepo) FindByStream(ctx context.Context, app, stream, vhost string) (*model.MediaStream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastFindCtxErr = ctx.Err()
	item := m.streams[stream]
	return item, nil
}

func (m *mockStreamRepo) Create(ctx context.Context, item *model.MediaStream) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streams[item.ZLMStream] = item
	return nil
}

func (m *mockStreamRepo) UpdateStatus(ctx context.Context, id, status string) error {
	m.mu.Lock()
	m.lastUpdateCtxErr = ctx.Err()
	m.mu.Unlock()
	if m.updateCh != nil {
		select {
		case m.updateCh <- struct{}{}:
		default:
		}
	}
	return nil
}

func setupTestSM() *StreamManager {
	mockRepo := newMockDeviceRepo()
	mockStreamRepo := newMockStreamRepo()
	mockClient := &MockEngineClient{}
	logger := zap.NewNop()
	return NewStreamManager(mockClient, mockRepo, mockStreamRepo, logger)
}

type failingStartEngineClient struct {
	*MockEngineClient
}

func (c *failingStartEngineClient) StartStream(context.Context, StreamStartRequest) (StreamInfo, error) {
	return StreamInfo{}, errors.New("start stream failed")
}

func TestAcquireAndReleaseRefCount(t *testing.T) {
	sm := setupTestSM()
	ctx := context.Background()

	// 注册测试设备
	devID := "test-device-1"
	_ = sm.deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: devID},
		RtspURL:   "rtsp://192.168.1.100:554/stream1",
	})

	// Acquire "play"
	err := sm.Acquire(ctx, devID, "play", nil)
	assert.NoError(t, err)

	state := sm.GetStream(ctx, devID)
	assert.NotNil(t, state)
	assert.Equal(t, int32(1), state.RefCount.Load())
	assert.Equal(t, "active", state.Status)

	// Acquire "infer" (第二次引用)
	err = sm.Acquire(ctx, devID, "infer", nil)
	assert.NoError(t, err)

	state = sm.GetStream(ctx, devID)
	assert.Equal(t, int32(2), state.RefCount.Load())

	// Release "play"
	err = sm.Release(ctx, devID, "play")
	assert.NoError(t, err)

	state = sm.GetStream(ctx, devID)
	assert.Equal(t, int32(1), state.RefCount.Load())

	// Release "infer"
	err = sm.Release(ctx, devID, "infer")
	assert.NoError(t, err)

	state = sm.GetStream(ctx, devID)
	assert.Equal(t, int32(0), state.RefCount.Load())
	assert.Equal(t, "inactive", state.Status)
}

func TestResolveStreamRTSPURL(t *testing.T) {
	deviceRepo := newMockDeviceRepo()
	manager := NewStreamManager(&MockEngineClient{}, deviceRepo, newMockStreamRepo(), zap.NewNop())
	ctx := context.Background()
	require.NoError(t, deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: "camera-1"},
		RtspURL:   "  rtsp://camera.local/stream1  ",
	}))

	tests := []struct {
		name     string
		streamID string
		want     string
	}{
		{name: "main stream", streamID: "camera-1", want: "rtsp://camera.local/stream1"},
		{name: "sub stream", streamID: "camera-1_sub", want: "rtsp://camera.local/stream2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.resolveStreamRTSPURL(ctx, tt.streamID)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStreamManagerSeparatesSameDeviceAcrossNodes(t *testing.T) {
	sm := setupTestSM()
	ctx := context.Background()
	deviceID := "shared-device"
	assert.NoError(t, sm.deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: deviceID}, RtspURL: "rtsp://example/stream",
	}))

	routeA := StreamRoute{NodeID: "node-a", DeviceID: deviceID}
	routeB := StreamRoute{NodeID: "node-b", DeviceID: deviceID}
	assert.NoError(t, sm.AcquireOnNode(ctx, routeA, "infer:task-a", map[string]string{"task_id": "task-a"}))
	assert.NoError(t, sm.AcquireOnNode(ctx, routeB, "infer:task-b", map[string]string{"task_id": "task-b"}))

	stateA := sm.GetStreamOnNode(ctx, routeA)
	stateB := sm.GetStreamOnNode(ctx, routeB)
	assert.NotNil(t, stateA)
	assert.NotNil(t, stateB)
	assert.NotSame(t, stateA, stateB)

	assert.NoError(t, sm.ReleaseOnNode(ctx, routeA, "infer:task-a"))
	assert.Equal(t, int32(0), stateA.RefCount.Load())
	assert.Equal(t, int32(1), stateB.RefCount.Load())
}

func TestRestoreInferenceOnNodeRollsBackConsumerWhenEngineStartFails(t *testing.T) {
	deviceRepo := newMockDeviceRepo()
	streamRepo := newMockStreamRepo()
	manager := NewStreamManager(&failingStartEngineClient{MockEngineClient: &MockEngineClient{}}, deviceRepo, streamRepo, zap.NewNop())
	ctx := context.Background()
	route := StreamRoute{NodeID: "node-a", DeviceID: "camera-a"}
	require.NoError(t, deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: route.DeviceID},
		RtspURL:   "rtsp://camera/live",
	}))

	err := manager.RestoreInferenceOnNode(ctx, route, "infer:task-a", map[string]string{"task_id": "task-a"})
	require.Error(t, err)
	state := manager.GetStreamOnNode(ctx, route)
	require.NotNil(t, state)
	assert.Equal(t, int32(0), state.RefCount.Load())
	_, exists := state.Consumers.Load("infer:task-a")
	assert.False(t, exists)
}

func TestKeepAlive(t *testing.T) {
	sm := setupTestSM()
	ctx := context.Background()

	devID := "test-device-2"
	_ = sm.deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: devID},
		RtspURL:   "rtsp://192.168.1.100:554/stream2",
	})

	_ = sm.Acquire(ctx, devID, "infer", nil)

	// 记录旧时间
	state := sm.GetStream(ctx, devID)
	var oldAlive time.Time
	state.Consumers.Range(func(key, value interface{}) bool {
		consumer := value.(*ConsumerInfo)
		oldAlive = consumer.LastAlive
		return false
	})

	time.Sleep(10 * time.Millisecond)
	sm.KeepAlive(ctx, devID, "infer")

	// 检查 LastAlive 已更新
	state.Consumers.Range(func(key, value interface{}) bool {
		consumer := value.(*ConsumerInfo)
		assert.True(t, consumer.LastAlive.After(oldAlive))
		return false
	})
}

func TestRetryBackoff(t *testing.T) {
	sm := setupTestSM()
	ctx := context.Background()

	devID := "test-device-3"
	_ = sm.deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: devID},
		RtspURL:   "rtsp://192.168.1.100:554/stream3",
	})

	_ = sm.Acquire(ctx, devID, "infer", nil)

	// 触发流找不到事件
	sm.HandleStreamNotFound(ctx, "live", devID, "__defaultVhost__")

	state := sm.GetStream(ctx, devID)
	assert.Equal(t, 1, state.RetryCount)
	assert.NotNil(t, state.RetryAt)

	// 触发多次直到上限
	for i := 0; i < 4; i++ {
		sm.HandleStreamNotFound(ctx, "live", devID, "__defaultVhost__")
	}

	state = sm.GetStream(ctx, devID)
	assert.Equal(t, 5, state.RetryCount)

	// 再触发一次应进入 error 状态
	sm.HandleStreamNotFound(ctx, "live", devID, "__defaultVhost__")
	state = sm.GetStream(ctx, devID)
	assert.Equal(t, "error", state.Status)
}

func TestListStreams(t *testing.T) {
	sm := setupTestSM()
	ctx := context.Background()

	// 创建设备并启动流
	dev1 := "dev-list-1"
	dev2 := "dev-list-2"
	_ = sm.deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: dev1},
		RtspURL:   "rtsp://192.168.1.100:554/s1",
	})
	_ = sm.deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: dev2},
		RtspURL:   "rtsp://192.168.1.101:554/s2",
	})

	_ = sm.Acquire(ctx, dev1, "play", nil)
	_ = sm.Acquire(ctx, dev2, "infer", nil)

	streams := sm.ListStreams(ctx)
	assert.Equal(t, 2, len(streams))

	_ = sm.Release(ctx, dev1, "play")
	_ = sm.Release(ctx, dev2, "infer")
}

func TestReleaseUsesBackgroundContextForAsyncStatusUpdate(t *testing.T) {
	deviceRepo := newMockDeviceRepo()
	streamRepo := newMockStreamRepo()
	streamRepo.updateCh = make(chan struct{}, 1)
	mockClient := &MockEngineClient{}
	sm := NewStreamManager(mockClient, deviceRepo, streamRepo, zap.NewNop())

	ctx := context.Background()
	devID := "dev-release-canceled-ctx"
	_ = sm.deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: devID},
		RtspURL:   "rtsp://192.168.1.100:554/stream",
	})

	err := sm.Acquire(ctx, devID, "detect", nil)
	assert.NoError(t, err)

	streamRepo.mu.Lock()
	streamRepo.streams[devID] = &model.MediaStream{BaseModel: model.BaseModel{ID: "stream-release-canceled-ctx"}, ZLMStream: devID}
	streamRepo.mu.Unlock()

	requestCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = sm.Release(requestCtx, devID, "detect")
	assert.NoError(t, err)

	select {
	case <-streamRepo.updateCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for async stream status update")
	}

	streamRepo.mu.Lock()
	findCtxErr := streamRepo.lastFindCtxErr
	updateCtxErr := streamRepo.lastUpdateCtxErr
	streamRepo.mu.Unlock()

	assert.NoError(t, findCtxErr)
	assert.NoError(t, updateCtxErr)
}

func TestListStreams_FilterInactive(t *testing.T) {
	sm := setupTestSM()
	ctx := context.Background()

	// 创建设备并 Acquire 后 Release
	devID := "dev-filter-test"
	_ = sm.deviceRepo.Create(ctx, &model.Device{
		BaseModel: model.BaseModel{ID: devID},
		RtspURL:   "rtsp://192.168.1.100:554/stream",
	})

	err := sm.Acquire(ctx, devID, "detect", nil)
	assert.NoError(t, err)

	// Release 后应该变成 inactive
	err = sm.Release(ctx, devID, "detect")
	assert.NoError(t, err)

	// ListStreams 应该过滤掉无引用的 inactive 流
	streams := sm.ListStreams(ctx)
	assert.Equal(t, 0, len(streams))
}
