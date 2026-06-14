package ipc

import (
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"

	fbs "github.com/niko-admin/niko-admin/internal/pkg/ipc/fbs/aivision/ipc"
)

func TestFlatBuffersToInferenceResult(t *testing.T) {
	builder := flatbuffers.NewBuilder(1024)

	// Create strings
	taskIDStr := builder.CreateString("task-123")
	algoNameStr := builder.CreateString("face-detect")
	deviceIDStr := builder.CreateString("camera-456")
	alarmTypeStr := builder.CreateString("intrusion")
	alarmLevelStr := builder.CreateString("critical")
	snapshotStr := builder.CreateString("/path/to/snap.jpg")
	cropStr := builder.CreateString("/path/to/crop.jpg")
	bgStr := builder.CreateString("/path/to/bg.jpg")
	resultJSONStr := builder.CreateString(`{"test":true}`)

	// Construct InferenceResultMsg
	fbs.InferenceResultMsgStart(builder)
	fbs.InferenceResultMsgAddTaskId(builder, taskIDStr)
	fbs.InferenceResultMsgAddAlgoName(builder, algoNameStr)
	fbs.InferenceResultMsgAddDeviceId(builder, deviceIDStr)
	fbs.InferenceResultMsgAddFrameTsNs(builder, 1625000000000000000)
	fbs.InferenceResultMsgAddFrameWidth(builder, 1920)
	fbs.InferenceResultMsgAddFrameHeight(builder, 1080)
	fbs.InferenceResultMsgAddRecordType(builder, fbs.RecordTypeAlarm)
	fbs.InferenceResultMsgAddAlarmType(builder, alarmTypeStr)
	fbs.InferenceResultMsgAddAlarmLevel(builder, alarmLevelStr)
	fbs.InferenceResultMsgAddSnapshotPath(builder, snapshotStr)
	fbs.InferenceResultMsgAddCropPath(builder, cropStr)
	fbs.InferenceResultMsgAddBackgroundPath(builder, bgStr)
	fbs.InferenceResultMsgAddResultJson(builder, resultJSONStr)
	fbs.InferenceResultMsgAddInferTimeUs(builder, 1500)

	msgOffset := fbs.InferenceResultMsgEnd(builder)
	fbs.FinishInferenceResultMsgBuffer(builder, msgOffset)

	buf := builder.FinishedBytes()

	// Parse
	params := FlatBuffersToInferenceResult(buf)
	assert.NotNil(t, params)
	assert.Equal(t, "task-123", params.TaskID)
	assert.Equal(t, "face-detect", params.AlgoName)
	assert.Equal(t, "camera-456", params.DeviceID)
	assert.Equal(t, uint64(1625000000000000000), params.FrameTS)
	assert.Equal(t, 1920, params.FrameWidth)
	assert.Equal(t, 1080, params.FrameHeight)
	assert.Equal(t, "alarm", params.RecordType)
	assert.Equal(t, "intrusion", params.AlarmType)
	assert.Equal(t, "critical", params.AlarmLevel)
	assert.Equal(t, "/path/to/snap.jpg", params.SnapshotPath)
	assert.Equal(t, "/path/to/crop.jpg", params.CropPath)
	assert.Equal(t, "/path/to/bg.jpg", params.BackgroundPath)
	assert.Equal(t, `{"test":true}`, params.ResultJSON)
	assert.Equal(t, uint32(1500), params.InferTimeUS)
}
