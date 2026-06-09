// Package ipc 提供 Go 控制面与 C++ 数据面之间的 IPC 通信支持。
// 包含 FlatBuffers 与 Go model 之间的类型转换、IPC 通信客户端等。
package ipc

import (
	"encoding/json"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"go.uber.org/zap"
	"gorm.io/datatypes"

	"github.com/niko-admin/niko-admin/internal/model"
	fbs "github.com/niko-admin/niko-admin/internal/pkg/ipc/fbs/aivision/ipc"
)

// ============================================================
// JSON 结构定义 (Go model 的 JSONB 字段格式)
// ============================================================

// PointJSON 表示像素坐标点。
type PointJSON struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// PolygonJSON 表示多边形区域 (顶点列表)。
type PolygonJSON struct {
	Points []PointJSON `json:"points"`
}

// RegionJSONList 是多边形区域列表。
type RegionJSONList = []PolygonJSON

// BoundingBoxJSON 表示检测框。
type BoundingBoxJSON struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	W          int     `json:"w"`
	H          int     `json:"h"`
	Confidence float32 `json:"confidence"`
	LabelID    int     `json:"label_id"`
	LabelName  string  `json:"label_name,omitempty"`
	TrackID    int     `json:"track_id,omitempty"`
}

// InferenceResultJSON 表示推理结果的 JSON 扩展字段。
type InferenceResultJSON struct {
	Detections []BoundingBoxJSON `json:"detections,omitempty"`
	AlarmType  string            `json:"alarm_type,omitempty"`
	AlarmLevel string            `json:"alarm_level,omitempty"`
	Extra      map[string]any    `json:"extra,omitempty"`
}

// ============================================================
// 转换函数: JSON ↔ FlatBuffers (区域数据)
// ============================================================

// RegionJSONToFlatBuffers TODO: flatc 生成 Go 代码后实现 FlatBuffers 序列化。
func RegionJSONToFlatBuffers(jsonData datatypes.JSON) []byte {
	if len(jsonData) == 0 {
		return nil
	}

	var regions RegionJSONList
	if err := json.Unmarshal(jsonData, &regions); err != nil {
		zap.L().Warn("IPC: 解析区域 JSON 失败, 跳过",
			zap.ByteString("json", jsonData),
			zap.Error(err))
		return nil
	}

	if len(regions) == 0 {
		return nil
	}

	zap.L().Debug("IPC: RegionJSONToFlatBuffers 待 flatc 生成代码后实现")
	return nil
}

// FlatBuffersToRegionJSON TODO: flatc 生成代码后实现 FlatBuffers 读取。
func FlatBuffersToRegionJSON(fbData []byte) datatypes.JSON {
	if len(fbData) == 0 {
		return datatypes.JSON("[]")
	}

	zap.L().Debug("IPC: FlatBuffersToRegionJSON 待 flatc 生成代码后实现")
	return datatypes.JSON("[]")
}

// ============================================================
// 转换函数: Go InferTask → C++ StartStreamCmd
// ============================================================

type StartStreamParams struct {
	TaskID            string              `json:"task_id"`
	StreamURL         string              `json:"stream_url"`
	DecodeHWType      int                 `json:"decode_hw_type"`
	DeviceID          string              `json:"device_id"`
	DeviceName        string              `json:"device_name"`
	EnableInfer       bool                `json:"enable_infer"`
	EnablePlayback    bool                `json:"enable_playback"`
	MaxReconnects     int                 `json:"max_reconnects"`
	ReconnectInterval int                 `json:"reconnect_interval"`
	Algorithms        []AlgoBindingParams `json:"algorithms"`
}

type AlgoBindingParams struct {
	AlgoName        string
	AlgoVersion     string
	PackageID       string
	SoPath          string
	AlgoParamsJSON  string
	ROIRegions      datatypes.JSON
	MarkRegions     datatypes.JSON
	LineRegions     datatypes.JSON
	ScheduleEnabled bool
	ScheduleStart   string
	ScheduleEnd     string
	SortOrder       int
	LabelMaps       []LabelMapEntry
}

type LabelMapEntry struct {
	LabelID      int
	CategoryCode int
	DisplayName  string
}

// InferTaskToStartStream 将 Go model.InferTask 转换为 StartStreamParams。
func InferTaskToStartStream(task *model.InferTask, algorithms []model.InferTaskAlgorithm, labelMaps map[string][]model.AlgorithmLabelMap) *StartStreamParams {
	if task == nil {
		return nil
	}

	params := &StartStreamParams{
		TaskID:            task.ID,
		StreamURL:         task.StreamURL,
		DecodeHWType:      task.DecodeHWType,
		DeviceID:          task.DeviceID,
		MaxReconnects:     task.MaxReconnects,
		ReconnectInterval: task.ReconnectInterval,
		Algorithms:        make([]AlgoBindingParams, 0, len(algorithms)),
	}

	for _, algo := range algorithms {
		binding := AlgoBindingParams{
			AlgoName:        algo.AlgoName,
			AlgoVersion:     algo.AlgoVersion,
			PackageID:       algo.PackageID,
			AlgoParamsJSON:  string(algo.AIParams),
			ROIRegions:      algo.ROIRegions,
			MarkRegions:     algo.MarkRegions,
			LineRegions:     algo.LineRegions,
			ScheduleEnabled: algo.ScheduleEnabled,
			ScheduleStart:   algo.ScheduleStart,
			ScheduleEnd:     algo.ScheduleEnd,
			SortOrder:       algo.SortOrder,
		}

		if maps, ok := labelMaps[algo.PackageID]; ok {
			binding.LabelMaps = make([]LabelMapEntry, 0, len(maps))
			for _, m := range maps {
				binding.LabelMaps = append(binding.LabelMaps, LabelMapEntry{
					LabelID:      m.LabelID,
					CategoryCode: m.CategoryCode,
					DisplayName:  m.DisplayName,
				})
			}
		}

		params.Algorithms = append(params.Algorithms, binding)
	}

	return params
}

// StartStreamParamsToFlatBuffers 将 StartStreamParams 序列化为 FlatBuffers 格式
func StartStreamParamsToFlatBuffers(params *StartStreamParams) []byte {
	if params == nil {
		return nil
	}

	// 使用 FlatBuffers 序列化
	// TODO: 实际序列化需要 flatc 生成的 StreamStartCmd 构建器
	// 先用 JSON 透传
	jsonBytes, _ := json.Marshal(params)
	return jsonBytes
}

// ============================================================
// 转换函数: C++ InferenceResultMsg → Go SmartRecord
// ============================================================

// InferenceResultParams 表示从 C++ 接收的推理结果参数。
type InferenceResultParams struct {
	TaskID         string
	AlgoName       string
	DeviceID       string
	FrameTS        uint64
	FrameWidth     int
	FrameHeight    int
	Detections     []BoundingBoxJSON
	RecordType     string
	AlarmType      string
	AlarmLevel     string
	TriggeredIDs   []string
	SnapshotPath   string
	CropPath       string
	BackgroundPath string
	ResultJSON     string
	InferTimeUS    uint32
}

// FlatBuffersToInferenceResult TODO: flatc 生成代码后实现 FlatBuffers 读取。
func FlatBuffersToInferenceResult(fbData []byte) *InferenceResultParams {
	if len(fbData) == 0 {
		return nil
	}

	zap.L().Debug("IPC: FlatBuffersToInferenceResult 待 flatc 生成代码后实现")
	return nil
}

// InferenceResultToSmartRecord 将推理结果参数转换为 model.SmartRecord。
func InferenceResultToSmartRecord(params *InferenceResultParams, taskName, deviceName, algoVersion string) *model.SmartRecord {
	if params == nil {
		return nil
	}

	record := &model.SmartRecord{
		RecordType:         params.RecordType,
		CaptureTime:        nanosToTime(params.FrameTS),
		TaskID:             &params.TaskID,
		TaskName:           taskName,
		DeviceID:           &params.DeviceID,
		DeviceName:         deviceName,
		AlgorithmName:      params.AlgoName,
		AlgorithmVersion:   algoVersion,
		AlarmType:          params.AlarmType,
		AlarmLevel:         params.AlarmLevel,
		TriggeredLineIDs:   params.TriggeredIDs,
		SnapshotImageURL:   params.SnapshotPath,
		TargetCropURL:      params.CropPath,
		BackgroundImageURL: params.BackgroundPath,
	}

	if len(params.Detections) > 0 {
		maxConf := float64(0)
		for _, det := range params.Detections {
			if float64(det.Confidence) > maxConf {
				maxConf = float64(det.Confidence)
			}
		}
		record.Confidence = &maxConf
	}

	rawResult := InferenceResultJSON{
		Detections: params.Detections,
		AlarmType:  params.AlarmType,
		AlarmLevel: params.AlarmLevel,
	}
	if params.InferTimeUS > 0 {
		rawResult.Extra = map[string]any{
			"infer_time_us": params.InferTimeUS,
		}
	}
	rawBytes, err := json.Marshal(rawResult)
	if err != nil {
		zap.L().Warn("IPC: 序列化 RawResult 失败", zap.Error(err))
		record.RawResult = datatypes.JSON("{}")
	} else {
		record.RawResult = datatypes.JSON(rawBytes)
	}

	return record
}

// ============================================================
// 转换函数: C++ StreamStatusMsg → Go InferTask 状态更新
// ============================================================

// StreamStatusParams 表示流状态变更参数。
type StreamStatusParams struct {
	TaskID         string
	Status         string
	ErrorCode      string
	ErrorMessage   string
	ReconnectCount int
	MaxReconnects  int
	DeviceID       string
	PlayURL        string
}

// FlatBuffersToStreamStatus 解析 FlatBuffers StreamStatusRspMsg
func FlatBuffersToStreamStatus(fbData []byte) *StreamStatusParams {
	if len(fbData) == 0 {
		return nil
	}

	parseStatus := func(data []byte) *StreamStatusParams {
		status := fbs.GetRootAsStreamStatusRspMsg(data, 0)
		if status == nil {
			return nil
		}
		deviceID := string(status.DeviceId())
		playURL := string(status.PlaybackUrl())
		if deviceID == "" && playURL == "" && !status.IsRunning() {
			return nil
		}
		statusStr := "unknown"
		if status.IsRunning() {
			statusStr = "running"
		}
		return &StreamStatusParams{
			DeviceID: deviceID,
			PlayURL:  playURL,
			Status:   statusStr,
		}
	}

	// C++ 引擎当前直接返回 StreamStatusRspMsg，必须优先按直接响应解析；
	// 否则 direct table 可能被误识别为 IPCEnvelope，导致 payload 为空。
	if params := parseStatus(fbData); params != nil {
		return params
	}

	// 兼容带 IPCEnvelope 的异步状态上报。
	env := fbs.GetRootAsIPCEnvelope(fbData, 0)
	if env != nil && env.SignalType() == fbs.SignalTypeStreamStatusReport {
		payload := env.PayloadBytes()
		if len(payload) == 0 {
			return nil
		}
		return parseStatus(payload)
	}

	return nil
}

// ============================================================
// 转换函数: C++ AlgoLoadResultMsg → Go AlgorithmPackage 更新
// ============================================================

// AlgoLoadResultParams 表示算法加载结果参数。
type AlgoLoadResultParams struct {
	TaskID         string
	AlgoName       string
	AlgoVersion    string
	PackageID      string
	Success        bool
	SelfCheck      string
	ErrorCode      string
	ErrorMessage   string
	LoadTimeMS     uint32
	NPUMemoryBytes uint64
}

// FlatBuffersToAlgoLoadResult 从 FlatBuffers 字节数组反序列化算法自检结果。
func FlatBuffersToAlgoLoadResult(fbData []byte) *AlgoLoadResultParams {
	if len(fbData) == 0 {
		return nil
	}

	msg := fbs.GetRootAsAlgoLoadResultMsg(fbData, 0)
	return &AlgoLoadResultParams{
		TaskID:         string(msg.TaskId()),
		AlgoName:       string(msg.AlgoName()),
		AlgoVersion:    string(msg.AlgoVersion()),
		PackageID:      string(msg.PackageId()),
		Success:        msg.Success(),
		SelfCheck:      msg.SelfCheckStatus().String(),
		ErrorCode:      string(msg.ErrorCode()),
		ErrorMessage:   string(msg.ErrorMessage()),
		LoadTimeMS:     msg.LoadTimeMs(),
		NPUMemoryBytes: msg.NpuMemoryBytes(),
	}
}

// ============================================================
// 转换函数: C++ EngineMetricsMsg → Go 监控指标
// ============================================================

// StreamMetricsSnapshot 表示单流指标快照。
type StreamMetricsSnapshot struct {
	TaskID             string
	QueueDepth         uint8
	EvictCount         uint32
	LastInferLatencyUS uint32
	AvgInferLatencyUS  uint32
	P95InferLatencyUS  uint32
	FrameCount         uint64
	LastFrameTS        uint64
}

// EngineMetricsSnapshot 表示引擎全局指标快照。
type EngineMetricsSnapshot struct {
	Streams           []StreamMetricsSnapshot
	ActiveStreamCount uint32
	DMAUsedBytes      uint64
	DMATotalBytes     uint64
	NPUUsedBytes      uint64
	NPUTotalBytes     uint64
	WorkerCount       uint32
	IdleWorkerCount   uint32
	TimestampNS       uint64
}

// FlatBuffersToEngineMetrics TODO: flatc 生成代码后实现。
func FlatBuffersToEngineMetrics(fbData []byte) *EngineMetricsSnapshot {
	if len(fbData) == 0 {
		return nil
	}

	zap.L().Debug("IPC: FlatBuffersToEngineMetrics 待 flatc 生成代码后实现")
	return nil
}

// ============================================================
// 辅助函数
// ============================================================

// nanosToTime 将 Unix 纳秒时间戳转换为 time.Time。
func nanosToTime(nanos uint64) time.Time {
	sec := int64(nanos / 1e9)
	nsec := int64(nanos % 1e9)
	return time.Unix(sec, nsec)
}

// StartSelfCheckCmdToFlatBuffers 将自检参数序列化为 FlatBuffers 格式。
func StartSelfCheckCmdToFlatBuffers(downloadURL, token, algoName, version string) []byte {
	builder := flatbuffers.NewBuilder(1024)

	// 创建字符串偏移量
	downloadURLOffset := builder.CreateString(downloadURL)
	tokenOffset := builder.CreateString(token)
	algoNameOffset := builder.CreateString(algoName)
	versionOffset := builder.CreateString(version)

	// 构建 StartSelfCheckCmd 对象
	fbs.StartSelfCheckCmdStart(builder)
	fbs.StartSelfCheckCmdAddDownloadUrl(builder, downloadURLOffset)
	fbs.StartSelfCheckCmdAddToken(builder, tokenOffset)
	fbs.StartSelfCheckCmdAddAlgorithmName(builder, algoNameOffset)
	fbs.StartSelfCheckCmdAddVersion(builder, versionOffset)
	offset := fbs.StartSelfCheckCmdEnd(builder)

	builder.Finish(offset)
	return builder.FinishedBytes()
}
