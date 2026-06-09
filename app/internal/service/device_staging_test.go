package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
)

// mockStagingRepo 模拟 discoveredDeviceRepo
type mockStagingRepo struct {
	items map[string]*model.DiscoveredDevice
}

func newMockStagingRepo() *mockStagingRepo {
	return &mockStagingRepo{items: make(map[string]*model.DiscoveredDevice)}
}

func (m *mockStagingRepo) List(ctx context.Context, source, status, keyword string, page, pageSize int) ([]model.DiscoveredDevice, int64, error) {
	var results []model.DiscoveredDevice
	for _, item := range m.items {
		if status != "" && item.Status != status {
			continue
		}
		if source != "" && item.Source != source {
			continue
		}
		results = append(results, *item)
	}
	return results, int64(len(results)), nil
}

func (m *mockStagingRepo) Create(ctx context.Context, item *model.DiscoveredDevice) error {
	m.items[item.ID] = item
	return nil
}

func (m *mockStagingRepo) Upsert(ctx context.Context, item *model.DiscoveredDevice) error {
	// 简化的 upsert，不验证 MAC 逻辑
	for _, existing := range m.items {
		if existing.DeviceIP == item.DeviceIP && existing.Source == item.Source {
			existing.DeviceName = item.DeviceName
			existing.Manufacturer = item.Manufacturer
			return nil
		}
	}
	item.ID = "mock-id-" + item.DeviceIP
	m.items[item.ID] = item
	return nil
}

func (m *mockStagingRepo) BatchUpdateStatus(ctx context.Context, ids []string, status string) error {
	for _, id := range ids {
		if item, ok := m.items[id]; ok {
			item.Status = status
		}
	}
	return nil
}

func (m *mockStagingRepo) FindByID(ctx context.Context, id string) (*model.DiscoveredDevice, error) {
	item := m.items[id]
	return item, nil
}

func (m *mockStagingRepo) MarkImported(ctx context.Context, id, deviceID string) error {
	if item, ok := m.items[id]; ok {
		item.Status = model.StatusImported
		item.MatchedDeviceID = &deviceID
	}
	return nil
}

func (m *mockStagingRepo) ResetByDeviceID(ctx context.Context, deviceID string) error {
	for _, item := range m.items {
		if item.MatchedDeviceID != nil && *item.MatchedDeviceID == deviceID && item.Status == model.StatusImported {
			item.Status = model.StatusPending
			item.ImportedAt = nil
			item.MatchedDeviceID = nil
		}
	}
	return nil
}

func TestDeviceStaging_Upsert(t *testing.T) {
	_ = zap.NewNop()
	repo := newMockStagingRepo()
	devRepo := newMockDeviceRepo()
	svc := NewDeviceStagingService(repo, devRepo)
	ctx := context.Background()

	// 第一次添加
	item := &model.DiscoveredDevice{
		Source:       model.SourceONVIF,
		DeviceIP:     "192.168.1.100",
		DeviceMAC:    "00:11:22:33:44:55",
		Manufacturer: "Hikvision",
		Model:        "DS-2CD2T4",
		Status:       model.StatusPending,
	}
	err := svc.AddDiscovered(ctx, item)
	assert.NoError(t, err)
}

func TestDeviceStaging_BatchOptions(t *testing.T) {
	repo := newMockStagingRepo()
	devRepo := newMockDeviceRepo()
	svc := NewDeviceStagingService(repo, devRepo)
	ctx := context.Background()

	// 添加两个设备作为发现记录
	dev1 := &model.DiscoveredDevice{Source: model.SourceONVIF, DeviceIP: "10.0.0.1", Status: model.StatusPending}
	dev2 := &model.DiscoveredDevice{Source: model.SourceONVIF, DeviceIP: "10.0.0.2", Status: model.StatusPending}
	_ = repo.Create(ctx, dev1)
	_ = repo.Create(ctx, dev2)

	// 批量忽略
	err := svc.BatchIgnore(ctx, []string{dev1.ID, dev2.ID})
	assert.NoError(t, err)

	item1, _ := repo.FindByID(ctx, dev1.ID)
	assert.Equal(t, model.StatusIgnored, item1.Status)
}
