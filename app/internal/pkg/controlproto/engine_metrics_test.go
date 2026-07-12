package controlproto

import (
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	fbs "github.com/niko-admin/niko-admin/internal/pkg/controlproto/fbs/aivision/control"
)

func TestFlatBuffersToEngineMetricsParsesMediaCapacityUsage(t *testing.T) {
	payload := buildEngineMetricsPayload(t)
	assertEngineMetricsSnapshot(t, FlatBuffersToEngineMetrics(payload))
}

func TestFlatBuffersToEngineMetricsParsesEnvelope(t *testing.T) {
	payload := buildEngineMetricsPayload(t)
	builder := flatbuffers.NewBuilder(512)
	payloadOffset := builder.CreateByteVector(payload)
	fbs.ControlEnvelopeStart(builder)
	fbs.ControlEnvelopeAddSchemaVersion(builder, 102)
	fbs.ControlEnvelopeAddSignalType(builder, fbs.SignalTypeEngineMetrics)
	fbs.ControlEnvelopeAddPayload(builder, payloadOffset)
	offset := fbs.ControlEnvelopeEnd(builder)
	builder.Finish(offset)
	assertEngineMetricsSnapshot(t, FlatBuffersToEngineMetrics(builder.FinishedBytes()))
}

func buildEngineMetricsPayload(t *testing.T) []byte {
	t.Helper()
	builder := flatbuffers.NewBuilder(256)
	fbs.EngineMetricsMsgStart(builder)
	fbs.EngineMetricsMsgAddActiveStreamCount(builder, 4)
	fbs.EngineMetricsMsgAddTimestampNs(builder, 12345)
	fbs.EngineMetricsMsgAddDecodeSessions(builder, 3)
	fbs.EngineMetricsMsgAddEncodeSessions(builder, 2)
	fbs.EngineMetricsMsgAddDecodeSlotsUsed(builder, 3)
	fbs.EngineMetricsMsgAddEncodeSlotsUsed(builder, 1)
	fbs.EngineMetricsMsgAddEgressBps(builder, 8_000_000)
	fbs.EngineMetricsMsgAddPreviewPipelineCount(builder, 1)
	fbs.EngineMetricsMsgAddInferencePipelineCount(builder, 2)
	fbs.EngineMetricsMsgAddMixedPipelineCount(builder, 1)
	fbs.EngineMetricsMsgAddMediaMetricsValid(builder, true)
	fbs.EngineMetricsMsgAddPreviewCapacity(builder, 8)
	fbs.EngineMetricsMsgAddPreviewInUse(builder, 2)
	fbs.EngineMetricsMsgAddPreviewCapacityValid(builder, true)
	offset := fbs.EngineMetricsMsgEnd(builder)
	builder.Finish(offset)
	return builder.FinishedBytes()
}

func assertEngineMetricsSnapshot(t *testing.T, got *EngineMetricsSnapshot) {
	t.Helper()
	if got == nil {
		t.Fatal("snapshot is nil")
	}
	if got.ActiveStreamCount != 4 || got.TimestampNS != 12345 ||
		got.DecodeSessions != 3 || got.EncodeSessions != 2 ||
		got.DecodeSlotsUsed != 3 || got.EncodeSlotsUsed != 1 ||
		got.EgressBPS != 8_000_000 || got.PreviewPipelineCount != 1 ||
		got.InferencePipelineCount != 2 || got.MixedPipelineCount != 1 ||
		!got.MediaMetricsValid || got.PreviewCapacity != 8 || got.PreviewInUse != 2 ||
		!got.PreviewCapacityValid {
		t.Fatalf("unexpected snapshot: %#v", got)
	}
}
