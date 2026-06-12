package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestValidateDeviceCode 验证 20 位 GB28181 编码格式
func TestValidateDeviceCode(t *testing.T) {
	tests := []struct {
		name     string
		deviceID string
		wantErr  bool
	}{
		{"valid 20 digits", "34020000001320000001", false},
		{"valid all zeros", "00000000000000000000", false},
		{"too short", "3402000000132000000", true},
		{"too long", "340200000013200000012", true},
		{"contains letter", "3402000000132000000A", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SIPService{}
			err := s.ValidateDeviceCode(tt.deviceID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestParseCatalogueResponse 解析 MANSCDP 目录响应
func TestParseCatalogueResponse(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Response>
<CmdType>Catalog</CmdType>
<SN>1234</SN>
<DeviceID>34020000001320000001</DeviceID>
<SumNum>2</SumNum>
<DeviceList>
<Item>
<DeviceID>34020000001320000001</DeviceID>
<Name>Camera 01</Name>
<Manufacturer>Hikvision</Manufacturer>
<Model>DS-2CD</Model>
<Status>ON</Status>
<Parental>0</Parental>
</Item>
<Item>
<DeviceID>34020000001320000002</DeviceID>
<Name>Camera 02</Name>
<Manufacturer> Dahua </Manufacturer>
<Model>IPC-HDW</Model>
<Status>OFF</Status>
<Parental>0</Parental>
</Item>
</DeviceList>
</Response>`

	channels, err := ParseCatalogueResponse(xmlData)
	assert.NoError(t, err)
	assert.Len(t, channels, 2)
	assert.Equal(t, "34020000001320000001", channels[0].DeviceID)
	assert.Equal(t, "Camera 01", channels[0].Name)
	assert.Equal(t, "Hikvision", channels[0].Manufacturer)
	assert.Equal(t, "ON", channels[0].Status)
	assert.Equal(t, "OFF", channels[1].Status)
}

// TestParseCatalogueResponse_InvalidCmdType 错误命令类型返回错误
func TestParseCatalogueResponse_InvalidCmdType(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Response>
<CmdType>DeviceInfo</CmdType>
<SN>1234</SN>
</Response>`
	_, err := ParseCatalogueResponse(xmlData)
	assert.Error(t, err)
}

// TestParseCatalogueResponse_InvalidXML 无效 XML 返回错误
func TestParseCatalogueResponse_InvalidXML(t *testing.T) {
	_, err := ParseCatalogueResponse("not valid xml")
	assert.Error(t, err)
}

// TestParseAlarmResponse 解析告警通知
func TestParseAlarmResponse(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<Notify>
<CmdType>Alarm</CmdType>
<SN>5678</SN>
<Alarm>
<DeviceID>34020000001320000001</DeviceID>
<AlarmType>1</AlarmType>
<AlarmLevel>1</AlarmLevel>
<Time>2026-06-06T10:00:00</Time>
</Alarm>
</Notify>`

	alarm, err := ParseAlarmResponse(xmlData)
	assert.NoError(t, err)
	assert.NotNil(t, alarm)
	assert.Equal(t, "34020000001320000001", alarm.DeviceID)
	assert.Equal(t, "1", alarm.AlarmType)
	assert.Equal(t, "1", alarm.AlarmLevel)
	assert.Equal(t, "2026-06-06T10:00:00", alarm.AlarmTime)
}

// TestCatalogueQueryXML 构造目录查询指令
func TestCatalogueQueryXML(t *testing.T) {
	xml := CatalogueQueryXML("1234")
	assert.Contains(t, xml, "<CmdType>Catalog</CmdType>")
	assert.Contains(t, xml, "<SN>1234</SN>")
	assert.Contains(t, xml, "<DeviceID>all</DeviceID>")
}

// TestMapChannelStatus 状态映射
func TestMapChannelStatus(t *testing.T) {
	assert.Equal(t, "online", MapChannelStatus("ON"))
	assert.Equal(t, "offline", MapChannelStatus("OFF"))
	assert.Equal(t, "unknown", MapChannelStatus(""))
	assert.Equal(t, "unknown", MapChannelStatus("INVALID"))
}

// TestFormatPlaybackTime 时间格式
func TestFormatPlaybackTime(t *testing.T) {
	tm := time.Date(2026, 6, 6, 10, 30, 0, 0, time.UTC)
	assert.Equal(t, "2026-06-06T10:30:00", formatPlaybackTime(tm))
}

// TestStartPlayback 录像回放时间校验
func TestStartPlayback_TimeValidation(t *testing.T) {
	s := &SIPService{zlmClient: nil}
	start := time.Now()
	end := start.Add(-1 * time.Hour)
	_, err := s.StartPlayback(context.Background(), "34020000001320000001", "test", start, end)
	assert.Error(t, err)
}

// TestStopLiveStream_NilZLMClient zlmClient 未配置时返回错误
func TestStopLiveStream_NilZLMClient(t *testing.T) {
	s := &SIPService{zlmClient: nil}
	err := s.StopLiveStream(context.Background(), "device1", "stream1")
	assert.Error(t, err)
}

// TestStartLiveStream_NilZLMClient 实时预览同上
func TestStartLiveStream_NilZLMClient(t *testing.T) {
	s := &SIPService{zlmClient: nil}
	_, err := s.StartLiveStream(context.Background(), "34020000001320000001", "stream1")
	assert.Error(t, err)
}

// TestPlaybackControl_NilZLMClient 回放控制同上
func TestPlaybackControl_NilZLMClient(t *testing.T) {
	s := &SIPService{zlmClient: nil}
	err := s.PlaybackControl(context.Background(), "stream1", "scale", 2.0, 0)
	assert.Error(t, err)
	err = s.PlaybackControl(context.Background(), "stream1", "seek", 0, 1000)
	assert.Error(t, err)
	err = s.PlaybackControl(context.Background(), "stream1", "invalid", 0, 0)
	assert.Error(t, err)
}

// TestHandleAlarm_EmptyDeviceID 告警 deviceID 为空时返回错误
func TestHandleAlarm_EmptyDeviceID(t *testing.T) {
	s := &SIPService{}
	err := s.HandleAlarm(context.Background(), AlarmInfo{})
	assert.Error(t, err)
}

type mockTaskClient struct {
	enqueueCalled bool
	lastTaskType  string
}

func (m *mockTaskClient) Enqueue(ctx context.Context, taskType string, payload interface{}) error {
	m.enqueueCalled = true
	m.lastTaskType = taskType
	return nil
}

func (m *mockTaskClient) EnqueueWithID(ctx context.Context, taskType string, payload interface{}, taskID string) error {
	m.enqueueCalled = true
	m.lastTaskType = taskType
	return nil
}

func (m *mockTaskClient) RemovePending(ctx context.Context, taskType, entityID string) error {
	return nil
}

// TestHandleAlarm_Enqueue 验证告警被推入队列
func TestHandleAlarm_Enqueue(t *testing.T) {
	mockClient := &mockTaskClient{}
	s := &SIPService{
		taskClient: mockClient,
	}
	alarm := AlarmInfo{
		DeviceID:   "34020000001320000001",
		AlarmType:  "1",
		AlarmLevel: "1",
		AlarmTime:  "2026-06-06T10:00:00",
	}
	err := s.HandleAlarm(context.Background(), alarm)
	assert.NoError(t, err)
	assert.True(t, mockClient.enqueueCalled)
	assert.Equal(t, "alarm:dispatch", mockClient.lastTaskType)
}
