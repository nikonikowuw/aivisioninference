package service

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AIVisionTaskService 处理 AIVisionTask 业务逻辑
type AIVisionTaskService struct {
	aivisiontaskRepo     *repository.AIVisionTaskRepository
	aiTimeScheduleRepo   *repository.AITimeScheduleRepository
	algorithmPackageRepo *repository.AlgorithmPackageRepository
	deviceRepo           *repository.DeviceRepository
	sipSvc               *SIPService
	streamManager        *StreamManager
}

// NewAIVisionTaskService 创建新的 AIVisionTaskService
func NewAIVisionTaskService(
	aivisiontaskRepo *repository.AIVisionTaskRepository,
	aiTimeScheduleRepo *repository.AITimeScheduleRepository,
	algorithmPackageRepo *repository.AlgorithmPackageRepository,
	deviceRepo *repository.DeviceRepository,
	sipSvc *SIPService,
	streamManager *StreamManager,
) *AIVisionTaskService {
	svc := &AIVisionTaskService{
		aivisiontaskRepo:     aivisiontaskRepo,
		aiTimeScheduleRepo:   aiTimeScheduleRepo,
		algorithmPackageRepo: algorithmPackageRepo,
		deviceRepo:           deviceRepo,
		sipSvc:               sipSvc,
		streamManager:        streamManager,
	}
	if streamManager != nil {
		streamManager.Subscribe(svc.HandleDeviceEvent)
	}
	return svc
}

// validateDeviceForInference 校验设备是否可用于推理任务
func (s *AIVisionTaskService) validateDeviceForInference(ctx context.Context, deviceID string) error {
	device, err := s.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return apperrors.New(apperrors.ErrDeviceNotFound, "")
	}
	if !device.Enabled {
		return apperrors.New(apperrors.ErrDeviceDisabled, "")
	}
	if device.Status != model.DeviceStatusOnline {
		return apperrors.New(apperrors.ErrDeviceOffline, "")
	}
	// 校验 access_type 是否支持推理
	switch device.AccessType {
	case model.DeviceAccessTypeRTSP, model.DeviceAccessTypeGB28181, model.DeviceAccessTypeNVRChannel:
		// 支持的接入类型
	default:
		return apperrors.New(apperrors.ErrDeviceTypeInvalid, "")
	}
	return nil
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

			for _, w := range windows {
				st, _ := time.Parse("15:04", w.Start)
				et, _ := time.Parse("15:04", w.End)
				startMin := st.Hour()*60 + st.Minute()
				endMin := et.Hour()*60 + et.Minute()

				for m := startMin; m < endMin; m++ {
					minutes[m]++
				}
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
						reason := "errors.streamOffline"
						if _, ok := event.Metadata["error"].(string); ok {
							reason = "errors.streamError"
						} else if event.EventType == "offline" {
							reason = "errors.gb28181SourceOffline"
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

	// 校验设备通道状态
	if err := s.validateDeviceForInference(ctx, req.DeviceChannelID); err != nil {
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
		// 校验新设备通道状态
		if err := s.validateDeviceForInference(ctx, req.DeviceChannelID); err != nil {
			return err
		}
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
	if err := ctx.Err(); err != nil {
		return err
	}

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

// StartTask 拉起推理流并更新状态
func (s *AIVisionTaskService) StartTask(ctx context.Context, task *model.AIVisionTask) error {
	// 启动前再次校验设备状态，防止创建时在线但启动时已离线
	if err := s.validateDeviceForInference(ctx, task.DeviceChannelID); err != nil {
		task.Status = model.TaskStatusError
		task.ErrorReason = "errors.deviceNotAvailable"
		return s.aivisiontaskRepo.Update(ctx, task)
	}

	metadata := map[string]string{
		"task_id":         task.ID,
		"algo_package_id": task.AlgoPackageID,
		"target_node_id":  task.TargetNodeID,
	}
	if s.algorithmPackageRepo != nil {
		algoPackage, err := s.algorithmPackageRepo.FindByID(ctx, task.AlgoPackageID)
		if err != nil {
			task.Status = model.TaskStatusError
			task.ErrorReason = "errors.algorithmPackageNotFound"
			return s.aivisiontaskRepo.Update(ctx, task)
		}
		soPath, err := resolveRuntimeSoPath(algoPackage)
		if err != nil {
			zap.L().Warn("resolve algorithm runtime so failed",
				zap.String("task_id", task.ID),
				zap.String("algo_package_id", task.AlgoPackageID),
				zap.Error(err),
			)
			task.Status = model.TaskStatusError
			task.ErrorReason = "errors.algorithmLoadFailed"
			return s.aivisiontaskRepo.Update(ctx, task)
		}
		metadata["algo_name"] = algoPackage.AlgorithmName
		metadata["algo_version"] = algoPackage.Version
		metadata["so_path"] = soPath
		metadata["algo_params_json"] = string(task.AIParams)
		zap.L().Info("resolved algorithm runtime library",
			zap.String("task_id", task.ID),
			zap.String("algo_package_id", task.AlgoPackageID),
			zap.String("algo_name", algoPackage.AlgorithmName),
			zap.String("so_path", soPath),
		)
	}

	if s.streamManager != nil {
		if err := s.streamManager.Acquire(ctx, task.DeviceChannelID, "infer", metadata); err != nil {
			zap.L().Warn("start ai vision stream failed",
				zap.String("task_id", task.ID),
				zap.String("device_channel_id", task.DeviceChannelID),
				zap.Error(err),
			)
			task.Status = model.TaskStatusError
			task.ErrorReason = "errors.connectionFailed"
			return s.aivisiontaskRepo.Update(ctx, task)
		}
	}

	task.Status = model.TaskStatusRunning
	task.ErrorReason = ""
	return s.aivisiontaskRepo.Update(ctx, task)
}

// StopTask 停止推理流并更新状态
func (s *AIVisionTaskService) StopTask(ctx context.Context, task *model.AIVisionTask, errorReason string) error {
	if s.streamManager != nil {
		_ = s.streamManager.Release(ctx, task.DeviceChannelID, "infer")
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

// RestartTask 释放旧推理流并重新拉起任务。
func (s *AIVisionTaskService) RestartTask(ctx context.Context, id string) error {
	task, err := s.aivisiontaskRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if s.streamManager != nil {
		_ = s.streamManager.Release(ctx, task.DeviceChannelID, "infer")
	}
	task.Status = model.TaskStatusReady
	task.ErrorReason = ""
	if err := s.aivisiontaskRepo.Update(ctx, task); err != nil {
		return err
	}
	return s.StartTask(ctx, task)
}

func resolveRuntimeSoPath(algoPackage *model.AlgorithmPackage) (string, error) {
	if algoPackage == nil {
		return "", fmt.Errorf("algorithm package is nil")
	}
	if filepath.IsAbs(algoPackage.SoPath) {
		if info, err := os.Stat(algoPackage.SoPath); err == nil && !info.IsDir() {
			return algoPackage.SoPath, nil
		}
	}

	packagePath := strings.TrimSpace(algoPackage.ExtractPath)
	if packagePath == "" {
		packagePath = strings.TrimSpace(algoPackage.PackagePath)
	}
	if packagePath == "" {
		return "", fmt.Errorf("algorithm package path is empty")
	}
	if info, err := os.Stat(packagePath); err != nil || info.IsDir() {
		// Fallback for relative PackagePath without uploads/ prefix
		if !strings.HasPrefix(packagePath, "uploads/") {
			fallbackPath := filepath.Join("uploads", packagePath)
			if info, err := os.Stat(fallbackPath); err == nil && !info.IsDir() {
				packagePath = fallbackPath
			}
		}
		// Check again after fallback attempt
		if info, err := os.Stat(packagePath); err != nil || info.IsDir() {
			return "", fmt.Errorf("algorithm package tar is unavailable: %s", packagePath)
		}
	}

	extractDir := strings.TrimSuffix(packagePath, filepath.Ext(packagePath)) + "_runtime"
	if soPath, err := findSoFile(extractDir); err == nil {
		return soPath, nil
	}
	if err := extractTarSecure(packagePath, extractDir); err != nil {
		return "", err
	}
	return findSoFile(extractDir)
}

// ResolveRuntimeSoPath returns the runtime .so path for an uploaded algorithm package.
func ResolveRuntimeSoPath(algoPackage *model.AlgorithmPackage) (string, error) {
	return resolveRuntimeSoPath(algoPackage)
}

func extractTarSecure(tarPath, extractDir string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}
	cleanRoot, err := filepath.Abs(extractDir)
	if err != nil {
		return err
	}

	reader := tar.NewReader(file)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		cleanName := filepath.Clean(header.Name)
		if filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, "..") || strings.Contains(cleanName, string(filepath.Separator)+".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe tar entry: %s", header.Name)
		}
		targetPath := filepath.Join(cleanRoot, cleanName)
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			return err
		}
		if absTarget != cleanRoot && !strings.HasPrefix(absTarget, cleanRoot+string(filepath.Separator)) {
			return fmt.Errorf("tar entry escapes extraction dir: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(absTarget, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(absTarget), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(absTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode)&0755)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, reader)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
	return nil
}

func findSoFile(root string) (string, error) {
	var soPath string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !entry.IsDir() && filepath.Ext(path) == ".so" {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			soPath = absPath
			return filepath.SkipAll
		}
		return nil
	}); err != nil {
		return "", err
	}
	if soPath == "" {
		return "", fmt.Errorf("no .so file found under %s", root)
	}
	return soPath, nil
}
