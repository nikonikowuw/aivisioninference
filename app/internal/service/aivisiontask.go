package service

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/datatypes"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AIVisionTaskService 处理 AIVisionTask 业务逻辑
type AIVisionTaskService struct {
	aivisiontaskRepo   *repository.AIVisionTaskRepository
	aiTimeScheduleRepo *repository.AITimeScheduleRepository
	sipSvc             *SIPService
	streamManager      *StreamManager
}

// NewAIVisionTaskService 创建新的 AIVisionTaskService
func NewAIVisionTaskService(
	aivisiontaskRepo *repository.AIVisionTaskRepository,
	aiTimeScheduleRepo *repository.AITimeScheduleRepository,
	sipSvc *SIPService,
	streamManager *StreamManager,
) *AIVisionTaskService {
	svc := &AIVisionTaskService{
		aivisiontaskRepo:   aivisiontaskRepo,
		aiTimeScheduleRepo: aiTimeScheduleRepo,
		sipSvc:             sipSvc,
		streamManager:      streamManager,
	}
	if streamManager != nil {
		streamManager.Subscribe(svc.HandleDeviceEvent)
	}
	return svc
}

// resolveSchedule 根据 schedule_id 加载时间配置并返回解析后的日期和时间窗
func (s *AIVisionTaskService) resolveSchedule(ctx context.Context, scheduleID string) (datatypes.Date, datatypes.Date, datatypes.JSON, error) {
	schedule, err := s.aiTimeScheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return datatypes.Date{}, datatypes.Date{}, nil, apperrors.New(apperrors.ErrAITimeScheduleNotFound, "")
	}
	return schedule.StartDate, schedule.EndDate, schedule.TimeWindows, nil
}

// CheckResourceConflict 检查给定节点和时间窗是否存在重叠冲突
func (s *AIVisionTaskService) CheckResourceConflict(ctx context.Context, nodeID string, startDate, endDate datatypes.Date, newWindows []dto.TimeWindow, excludeTaskID string) error {
	tasks, err := s.aivisiontaskRepo.FindActiveByNode(ctx, nodeID, excludeTaskID)
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}

	maxStreams := 4 // TODO: 从节点配置或 License 获取最大路数

	sd := time.Time(startDate)
	ed := time.Time(endDate)

	for current := sd; !current.After(ed); current = current.AddDate(0, 0, 1) {
		minutes := make([]int, 1440)
		var conflictNames []string

		for _, t := range tasks {
			tsd := time.Time(t.StartDate)
			ted := time.Time(t.EndDate)
			if current.Before(tsd) || current.After(ted) {
				continue
			}

			var windows []dto.TimeWindow
			if err := json.Unmarshal(t.TimeWindows, &windows); err != nil {
				continue
			}

			added := false
			for _, w := range windows {
				st, _ := time.Parse("15:04", w.Start)
				et, _ := time.Parse("15:04", w.End)
				startMin := st.Hour()*60 + st.Minute()
				endMin := et.Hour()*60 + et.Minute()

				for m := startMin; m < endMin; m++ {
					minutes[m]++
					if minutes[m] >= maxStreams {
						added = true
					}
				}
			}
			if added {
				conflictNames = append(conflictNames, t.Name)
			}
		}

		for _, w := range newWindows {
			st, err1 := time.Parse("15:04", w.Start)
			et, err2 := time.Parse("15:04", w.End)
			if err1 != nil || err2 != nil {
				return apperrors.New(apperrors.ErrTimeWindowFormat, "")
			}
			startMin := st.Hour()*60 + st.Minute()
			endMin := et.Hour()*60 + et.Minute()

			for m := startMin; m < endMin; m++ {
				if minutes[m]+1 > maxStreams {
					return apperrors.New(apperrors.ErrResourceConflict, "")
				}
			}
		}
	}
	return nil
}

func toJSONString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// HandleDeviceEvent 处理流管理器/引擎返回的设备和流事件
func (s *AIVisionTaskService) HandleDeviceEvent(ctx context.Context, event DeviceEvent) error {
	var items []model.AIVisionTask
	if err := s.aivisiontaskRepo.FindAllActive(ctx, &items); err == nil {
		for _, task := range items {
			if task.DeviceChannelID == event.DeviceID {
				if event.EventType == "online" {
					if task.Status != model.TaskStatusRunning {
						task.Status = model.TaskStatusRunning
						task.ErrorReason = ""
						_ = s.aivisiontaskRepo.Update(ctx, &task)
					}
				} else if event.EventType == "offline" || event.EventType == "error" {
					if task.Status == model.TaskStatusRunning {
						task.Status = model.TaskStatusError
						reason := "流离线"
						if r, ok := event.Metadata["error"].(string); ok {
							reason = "流异常: " + r
						} else if event.EventType == "offline" {
							reason = "GB28181 视频源掉线"
						}
						task.ErrorReason = reason
						_ = s.aivisiontaskRepo.Update(ctx, &task)
						_ = s.StopTask(ctx, &task, reason)
					}
				}
			}
		}
	}
	return nil
}

// List 分页查询推理任务
func (s *AIVisionTaskService) List(ctx context.Context, req dto.AIVisionTaskListRequest) ([]model.AIVisionTask, int64, error) {
	return s.aivisiontaskRepo.List(ctx, req)
}

// Create 创建推理任务（从时间配置复制日期/时间窗）
func (s *AIVisionTaskService) Create(ctx context.Context, req dto.CreateAIVisionTaskRequest) (*model.AIVisionTask, error) {
	startDate, endDate, timeWindows, err := s.resolveSchedule(ctx, req.ScheduleID)
	if err != nil {
		return nil, err
	}

	// 解析时间窗用于冲突检查
	var windows []dto.TimeWindow
	if err := json.Unmarshal(timeWindows, &windows); err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	if err := s.CheckResourceConflict(ctx, req.TargetNodeID, startDate, endDate, windows, ""); err != nil {
		return nil, err
	}

	item := &model.AIVisionTask{
		Name:            req.Name,
		Status:          model.TaskStatusReady,
		ScheduleID:      req.ScheduleID,
		DeviceChannelID: req.DeviceChannelID,
		AlgoPackageID:   req.AlgoPackageID,
		TargetNodeID:    req.TargetNodeID,
		StartDate:       startDate,
		EndDate:         endDate,
		TimeWindows:     timeWindows,
		AIParams:        datatypes.JSON(req.AIParams),
		ROIRegions:      datatypes.JSON(req.ROIRegions),
		MarkRegions:     datatypes.JSON(req.MarkRegions),
		LineRegions:     datatypes.JSON(req.LineRegions),
	}
	if err := s.aivisiontaskRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// GetByID 查询单个推理任务
func (s *AIVisionTaskService) GetByID(ctx context.Context, id string) (*model.AIVisionTask, error) {
	return s.aivisiontaskRepo.FindByID(ctx, id)
}

// Update 更新推理任务
func (s *AIVisionTaskService) Update(ctx context.Context, id string, req dto.UpdateAIVisionTaskRequest) error {
	item, err := s.aivisiontaskRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Status != "" {
		item.Status = req.Status
	}
	if req.DeviceChannelID != "" {
		item.DeviceChannelID = req.DeviceChannelID
	}
	if req.AlgoPackageID != "" {
		item.AlgoPackageID = req.AlgoPackageID
	}
	if req.TargetNodeID != "" {
		item.TargetNodeID = req.TargetNodeID
	}

	// 如果更新了时间配置，重新从配置复制日期/时间窗
	if req.ScheduleID != "" && req.ScheduleID != item.ScheduleID {
		startDate, endDate, timeWindows, err := s.resolveSchedule(ctx, req.ScheduleID)
		if err != nil {
			return err
		}
		item.ScheduleID = req.ScheduleID
		item.StartDate = startDate
		item.EndDate = endDate
		item.TimeWindows = timeWindows
	}

	// 解析时间窗用于冲突检查
	var windows []dto.TimeWindow
	if err := json.Unmarshal(item.TimeWindows, &windows); err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}

	if err := s.CheckResourceConflict(ctx, item.TargetNodeID, item.StartDate, item.EndDate, windows, id); err != nil {
		return err
	}

	if req.ErrorReason != "" {
		item.ErrorReason = req.ErrorReason
	}
	if req.AIParams != nil {
		item.AIParams = datatypes.JSON(req.AIParams)
	}
	if req.ROIRegions != nil {
		item.ROIRegions = datatypes.JSON(req.ROIRegions)
	}
	if req.MarkRegions != nil {
		item.MarkRegions = datatypes.JSON(req.MarkRegions)
	}
	if req.LineRegions != nil {
		item.LineRegions = datatypes.JSON(req.LineRegions)
	}
	return s.aivisiontaskRepo.Update(ctx, item)
}

// Delete 删除推理任务
func (s *AIVisionTaskService) Delete(ctx context.Context, id string) error {
	return s.aivisiontaskRepo.Delete(ctx, id)
}

// PatrolTasks 巡检任务，根据时间窗启动或停止推理流
func (s *AIVisionTaskService) PatrolTasks(ctx context.Context) error {
	var items []model.AIVisionTask
	if err := s.aivisiontaskRepo.FindAllActive(ctx, &items); err != nil {
		return err
	}

	now := time.Now()
	currentDate := now.Truncate(24 * time.Hour)
	currentMin := now.Hour()*60 + now.Minute()

	for _, task := range items {
		tsd := time.Time(task.StartDate).Truncate(24 * time.Hour)
		ted := time.Time(task.EndDate).Truncate(24 * time.Hour)

		inDateRange := !currentDate.Before(tsd) && !currentDate.After(ted)

		inTimeWindow := false
		if inDateRange {
			var windows []dto.TimeWindow
			if err := json.Unmarshal(task.TimeWindows, &windows); err == nil {
				for _, w := range windows {
					st, _ := time.Parse("15:04", w.Start)
					et, _ := time.Parse("15:04", w.End)
					startMin := st.Hour()*60 + st.Minute()
					endMin := et.Hour()*60 + et.Minute()
					if currentMin >= startMin && currentMin <= endMin {
						inTimeWindow = true
						break
					}
				}
			}
		}

		shouldRun := inDateRange && inTimeWindow

		if shouldRun && task.Status == model.TaskStatusReady {
			_ = s.StartTask(ctx, &task)
		} else if !shouldRun && task.Status == model.TaskStatusRunning {
			_ = s.StopTask(ctx, &task, "")
		}
	}
	return nil
}

// StartTask 拉起信令并更新状态
func (s *AIVisionTaskService) StartTask(ctx context.Context, task *model.AIVisionTask) error {
	streamID := "ai_task_" + task.ID
	if s.sipSvc != nil {
		_, err := s.sipSvc.StartLiveStream(ctx, task.DeviceChannelID, streamID)
		if err != nil {
			task.Status = model.TaskStatusError
			task.ErrorReason = "流拉起失败: " + err.Error()
			return s.aivisiontaskRepo.Update(ctx, task)
		}
	}

	task.Status = model.TaskStatusRunning
	task.ErrorReason = ""
	return s.aivisiontaskRepo.Update(ctx, task)
}

// StopTask 停止信令并更新状态
func (s *AIVisionTaskService) StopTask(ctx context.Context, task *model.AIVisionTask, errorReason string) error {
	streamID := "ai_task_" + task.ID
	if s.sipSvc != nil {
		_ = s.sipSvc.StopLiveStream(ctx, task.DeviceChannelID, streamID)
	}

	if errorReason != "" {
		task.Status = model.TaskStatusError
		task.ErrorReason = errorReason
	} else {
		task.Status = model.TaskStatusReady
		task.ErrorReason = ""
	}
	return s.aivisiontaskRepo.Update(ctx, task)
}
