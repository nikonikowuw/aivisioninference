// Package ipc 提供 Go 控制面与 C++ 数据面之间的 IPC 通信支持。
// 包含 FlatBuffers 与 Go model 之间的类型转换、UDS 通信客户端等。
package ipc

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"

	"github.com/niko-admin/niko-admin/internal/model"
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
	Detections []BoundingBoxJSON  `json:"detections,omitempty"`
	AlarmType  string             `json:"alarm_type,omitempty"`
	AlarmLevel string             `json:"alarm_level,omitempty"`
	Extra      map[string]any     `json:"extra,omitempty"`
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
	TaskID            string
	StreamURL         string
	DecodeHWType      int
	DeviceID          string
	DeviceName        string
	MaxReconnects     int
	ReconnectInterval int
	Algorithms        []AlgoBindingParams
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
			AlgoName:       algo.AlgoName,
			AlgoVersion:    algo.AlgoVersion,
			PackageID:      algo.PackageID,
			AlgoParamsJSON: string(algo.AIParams),
			ROIRegions:     algo.ROIRegions,
			MarkRegions:    algo.MarkRegions,
			LineRegions:    algo.LineRegions,
			ScheduleEnabled: algo.ScheduleEnabled,
			ScheduleStart:  algo.ScheduleStart,
			ScheduleEnd:    algo.ScheduleEnd,
			SortOrder:      algo.SortOrder,
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

// StartStreamParamsToFlatBuffers TODO: flatc 生成代码后实现 FlatBuffers 序列化。
func StartStreamParamsToFlatBuffers(params *StartStreamParams) []byte {
	if params == nil {
		return nil
	}

	zap.L().Debug("IPC: StartStreamParamsToFlatBuffers 待 flatc 生成代码后实现")
	return nil
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
		RecordType:        params.RecordType,
		CaptureTime:       nanosToTime(params.FrameTS),
		TaskID:            &params.TaskID,
		TaskName:          taskName,
		DeviceID:          &params.DeviceID,
		DeviceName:        deviceName,
		AlgorithmName:     params.AlgoName,
		AlgorithmVersion:  algoVersion,
		AlarmType:         params.AlarmType,
		AlarmLevel:        params.AlarmLevel,
		TriggeredLineIDs:  params.TriggeredIDs,
		SnapshotImageURL:  params.SnapshotPath,
		TargetCropURL:     params.CropPath,
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
}

// FlatBuffersToStreamStatus TODO: flatc 生成代码后实现。
func FlatBuffersToStreamStatus(fbData []byte) *StreamStatusParams {
	if len(fbData) == 0 {
		return nil
	}

	zap.L().Debug("IPC: FlatBuffersToStreamStatus 待 flatc 生成代码后实现")
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

// FlatBuffersToAlgoLoadResult TODO: flatc 生成代码后实现。
func FlatBuffersToAlgoLoadResult(fbData []byte) *AlgoLoadResultParams {
	if len(fbData) == 0 {
		return nil
	}

	zap.L().Debug("IPC: FlatBuffersToAlgoLoadResult 待 flatc 生成代码后实现")
	return nil
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

// recordTypeFromFB 将 FlatBuffers RecordType 枚举转换为 Go 字符串。
func recordTypeFromFB(fbType int) string {
	switch fbType {
	case 0:
		return model.RecordTypeCapture
	case 1:
		return model.RecordTypeRecognition
	case 2:
		return model.RecordTypeAlarm
	default:
		return model.RecordTypeCapture
	}
}

// streamStatusFromFB 将 FlatBuffers StreamStatus 枚举转换为 Go 字符串。
func streamStatusFromFB(fbStatus int) string {
	switch fbStatus {
	case 0:
		return model.InferTaskStatusPending
	case 1:
		return model.InferTaskStatusRunning
	case 2:
		return model.InferTaskStatusReconnecting
	case 3:
		return model.InferTaskStatusStopped
	case 4:
		return model.InferTaskStatusFailed
	default:
		return model.InferTaskStatusPending
	}
}

// selfCheckStatusFromFB 将 FlatBuffers SelfCheckStatus 枚举转换为 Go 字符串。
func selfCheckStatusFromFB(fbStatus int) string {
	switch fbStatus {
	case 0:
		return model.SelfCheckStatusPending
	case 1:
		return model.SelfCheckStatusRunning
	case 2:
		return model.SelfCheckStatusPassed
	case 3:
		return model.SelfCheckStatusFailed
	default:
		return model.SelfCheckStatusPending
	}
}

// alarmLevelToString 将告警级别标准化为 Go model 使用的字符串。
func alarmLevelToString(level int) string {
	switch level {
	case 0:
		return "info"
	case 1:
		return "warning"
	case 2:
		return "critical"
	default:
		return "info"
	}
}

// validateRegionJSON 校验区域 JSON 格式是否合法。
func validateRegionJSON(data datatypes.JSON) error {
	if len(data) == 0 {
		return nil
	}
	var regions RegionJSONList
	if err := json.Unmarshal(data, &regions); err != nil {
		return fmt.Errorf("invalid region JSON: %w", err)
	}
	for i, region := range regions {
		if len(region.Points) < 3 {
			return fmt.Errorf("region[%d] has %d points, minimum 3 required", i, len(region.Points))
		}
	}
	return nil
}
