package task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
)

// 告警派发任务类型
const (
	TypeAlarmDispatch = "alarm:dispatch"
)

// AlarmPayload 告警派发任务载荷
type AlarmPayload struct {
	SmartRecordID string `json:"smart_record_id"`
	DeviceID      string `json:"device_id"`
	DeviceName    string `json:"device_name"`
	AlgorithmName string `json:"algorithm_name"`
	AlarmType     string `json:"alarm_type"`
	AlarmLevel    string `json:"alarm_level"`
	CaptureTime   string `json:"capture_time"`
	SnapshotURL   string `json:"snapshot_url"`
	RawResult     string `json:"raw_result,omitempty"`
}

// AlarmDispatcher 告警派发器
// 将防抖判定后的有效告警推入 Asynq 队列，供 Webhook 等模块异步消费。
type AlarmDispatcher struct {
	client *asynq.Client
	logger *zap.Logger
}

// NewAlarmDispatcher 创建告警派发器
func NewAlarmDispatcher(client *asynq.Client) *AlarmDispatcher {
	return &AlarmDispatcher{
		client: client,
		logger: zap.L().With(zap.String("component", "alarm_dispatcher")),
	}
}

// Dispatch 将告警记录派发到 Asynq 队列
func (d *AlarmDispatcher) Dispatch(ctx context.Context, record *model.SmartRecord) error {
	payload := AlarmPayload{
		SmartRecordID: record.RecordID,
		DeviceID:      safeString(record.DeviceID),
		DeviceName:    record.DeviceName,
		AlgorithmName: record.AlgorithmName,
		AlarmType:     record.AlarmType,
		AlarmLevel:    record.AlarmLevel,
		CaptureTime:   record.CaptureTime.Format(time.RFC3339),
		SnapshotURL:   record.SnapshotImageURL,
		RawResult:     string(record.RawResult),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(TypeAlarmDispatch, data,
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
		asynq.Queue("alarm"),
	)

	info, err := d.client.Enqueue(task)
	if err != nil {
		d.logger.Error("failed to enqueue alarm dispatch task",
			zap.String("record_id", record.RecordID),
			zap.Error(err))
		return err
	}

	d.logger.Debug("alarm dispatch task enqueued",
		zap.String("record_id", record.RecordID),
		zap.String("task_id", info.ID))
	return nil
}

// RegisterAlarmHandler 注册告警派发任务处理器（在 Handler.RegisterHandlers 中调用）
func RegisterAlarmHandler(mux *asynq.ServeMux, alarmDispatcher *AlarmDispatcher) {
	mux.HandleFunc(TypeAlarmDispatch, func(ctx context.Context, t *asynq.Task) error {
		var payload AlarmPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return err
		}

		zap.L().Info("processing alarm dispatch",
			zap.String("record_id", payload.SmartRecordID),
			zap.String("alarm_type", payload.AlarmType),
			zap.String("alarm_level", payload.AlarmLevel))

		// TODO: 实际 Webhook 派发逻辑
		// 1. 查询 WebhookConfig 列表
		// 2. 匹配事件类型
		// 3. 发送 HTTP POST
		// 4. 记录 WebhookPushLog

		return nil
	})
}

// safeString 安全解引用字符串指针
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
