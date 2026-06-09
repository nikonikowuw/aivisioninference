// Package service 提供业务逻辑层实现,包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/csvx"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	"github.com/niko-admin/niko-admin/internal/pkg/i18n"
	"github.com/niko-admin/niko-admin/internal/pkg/zlm"
)

// deviceRepo 设备持久化接口
type deviceRepo interface {
	List(ctx context.Context, req dto.DeviceListRequest) ([]model.Device, int64, error)
	FindByID(ctx context.Context, id string) (*model.Device, error)
	FindByIDs(ctx context.Context, ids []string) ([]model.Device, error)
	ExistsByName(ctx context.Context, name, excludeID string) (bool, error)
	Create(ctx context.Context, item *model.Device) error
	Update(ctx context.Context, item *model.Device) error
	UpdateStatus(ctx context.Context, id, status, errorCode, errorMessage string) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error
	ListByGroupID(ctx context.Context, groupID string) ([]model.Device, error)
	ReplaceGroups(ctx context.Context, deviceID string, groupIDs []string) error
	FindByExternalKey(ctx context.Context, key string) (*model.Device, error)
}

// zlmClient ZLM API 客户端接口
type zlmClient interface {
	AddStreamProxy(ctx context.Context, req zlm.AddStreamProxyRequest) (string, error)
	CloseStream(ctx context.Context, req zlm.CloseStreamRequest) error
	IsMediaOnline(ctx context.Context, schema, vhost, app, stream string) (bool, error)
}

// deviceGroupRepo 设备分组持久化接口
type deviceGroupRepo interface {
	List(ctx context.Context, req dto.DeviceGroupListRequest) ([]model.DeviceGroup, int64, error)
	FindByID(ctx context.Context, id string) (*model.DeviceGroup, error)
	Create(ctx context.Context, item *model.DeviceGroup) error
	Update(ctx context.Context, item *model.DeviceGroup) error
	Delete(ctx context.Context, id string) error
	CountByGroupID(ctx context.Context, groupID string) (int64, error)
}

// cache 接口抽象,复用 internal/pkg/cache
type cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

// taskClient 异步任务客户端接口
type taskClient interface {
	Enqueue(ctx context.Context, taskType string, payload interface{}) error
}

// DeviceService 处理设备管理的业务逻辑
type DeviceService struct {
	deviceRepo           deviceRepo
	discoveredDeviceRepo discoveredDeviceRepo
	cache                cache
	taskClient           taskClient
	zlmClient            zlmClient
	streamManager        *StreamManager
}

// NewDeviceService 创建并返回一个新的 DeviceService 实例
func NewDeviceService(repo deviceRepo, discoveredDeviceRepo discoveredDeviceRepo, cache cache, taskClient taskClient, zlmClient zlmClient, streamManager *StreamManager) *DeviceService {
	return &DeviceService{
		deviceRepo:           repo,
		discoveredDeviceRepo: discoveredDeviceRepo,
		cache:                cache,
		taskClient:           taskClient,
		zlmClient:            zlmClient,
		streamManager:        streamManager,
	}
}

// getStatusCacheKey 获取设备状态缓存 Key
func (s *DeviceService) getStatusCacheKey(id string) string {
	return fmt.Sprintf("device:status:%s", id)
}

// List 返回分页的设备列表
func (s *DeviceService) List(ctx context.Context, req dto.DeviceListRequest) ([]dto.DeviceListResponse, int64, error) {
	items, total, err := s.deviceRepo.List(ctx, req)
	if err != nil {
		zap.L().Error("device list failed", zap.Error(err))
		return nil, 0, apperrors.New(apperrors.ErrInternal, "")
	}

	result := make([]dto.DeviceListResponse, len(items))
	for i, item := range items {
		result[i] = toDeviceListResponse(&item)
	}
	return result, total, nil
}

// GetByID 根据 ID 获取设备详情
func (s *DeviceService) GetByID(ctx context.Context, id string) (*dto.DeviceResponse, error) {
	item, err := s.deviceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrDeviceNotFound, "")
	}
	return toDeviceResponse(item), nil
}

// Create 创建设备,包含名称唯一性校验和密码加密
func (s *DeviceService) Create(ctx context.Context, req dto.DeviceCreateRequest) (*dto.DeviceResponse, error) {
	// 名称唯一性校验
	exists, err := s.deviceRepo.ExistsByName(ctx, req.DeviceName, "")
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if exists {
		return nil, apperrors.New(apperrors.ErrDeviceNameTaken, "")
	}

	// 条件必填校验
	if req.AccessType == "rtsp" && req.RtspURL == "" {
		return nil, apperrors.New(apperrors.ErrRTSPURLRequired, "")
	}
	if req.AccessType == "gb28181" && req.GB28181DeviceID == "" {
		return nil, apperrors.New(apperrors.ErrGB28181CodeRequired, "")
	}

	item := &model.Device{
		DeviceName:       req.DeviceName,
		AccessType:       req.AccessType,
		RtspURL:          req.RtspURL,
		GB28181DeviceID:  req.GB28181DeviceID,
		GB28181ChannelID: req.GB28181ChannelID,
		Username:         req.Username,
		Manufacturer:     req.Manufacturer,
		Model:            req.Model,
		FirmwareVersion:  req.FirmwareVersion,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		LocationDesc:     req.LocationDesc,
		Status:           model.DeviceStatusUnknown,
		Enabled:          true,
		Remark:           req.Remark,
	}

	// 设置 ExternalKey 用于唯一约束去重(rtsp:{url} | gb28181:{deviceID}:{channel})
	switch req.AccessType {
	case "rtsp":
		if req.RtspURL != "" {
			key := "rtsp:" + req.RtspURL
			item.ExternalKey = &key
		}
	case "gb28181":
		key := "gb28181:" + req.GB28181DeviceID + ":" + req.GB28181ChannelID
		item.ExternalKey = &key
	}

	// 密码加密
	if req.Password != "" {
		hashed, err := hash.Hash(req.Password)
		if err != nil {
			zap.L().Error("hash device password failed", zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		item.Password = hashed
	}

	if err := s.deviceRepo.Create(ctx, item); err != nil {
		zap.L().Error("create device failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	// 同步设备分组关联
	if len(req.GroupIDs) > 0 {
		if err := s.deviceRepo.ReplaceGroups(ctx, item.ID, req.GroupIDs); err != nil {
			zap.L().Error("sync device groups failed", zap.Error(err))
		}
	}

	// 重新加载分组信息以返回完整数据
	item, _ = s.deviceRepo.FindByID(ctx, item.ID)

	// 1. 缓存初始状态
	statusData, _ := json.Marshal(item.Status)
	_ = s.cache.Set(ctx, s.getStatusCacheKey(item.ID), statusData, 24*time.Hour)

	// 2. 发起异步连接探测任务(仅测试 RTSP 地址连通性)
	_ = s.taskClient.Enqueue(ctx, "device:detect", map[string]string{"id": item.ID})

	return toDeviceResponse(item), nil
}

// Update 更新设备,包含名称唯一性校验和可选密码更新
func (s *DeviceService) Update(ctx context.Context, id string, req dto.DeviceUpdateRequest) (*dto.DeviceResponse, error) {
	item, err := s.deviceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrDeviceNotFound, "")
	}

	// 名称唯一性校验
	if req.DeviceName != "" && req.DeviceName != item.DeviceName {
		exists, err := s.deviceRepo.ExistsByName(ctx, req.DeviceName, id)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		if exists {
			return nil, apperrors.New(apperrors.ErrDeviceNameTaken, "")
		}
		item.DeviceName = req.DeviceName
	}

	// 更新可编辑字段
	accessChanged := false
	if req.AccessType != "" && req.AccessType != item.AccessType {
		item.AccessType = req.AccessType
		accessChanged = true
	}
	if req.RtspURL != "" && req.RtspURL != item.RtspURL {
		item.RtspURL = req.RtspURL
		accessChanged = true
	}
	if req.GB28181DeviceID != "" && req.GB28181DeviceID != item.GB28181DeviceID {
		item.GB28181DeviceID = req.GB28181DeviceID
		accessChanged = true
	}
	if req.GB28181ChannelID != "" && req.GB28181ChannelID != item.GB28181ChannelID {
		item.GB28181ChannelID = req.GB28181ChannelID
		accessChanged = true
	}

	if accessChanged {
		// 重新生成 ExternalKey
		item.ExternalKey = nil // 先置空,再根据类型生成
		switch item.AccessType {
		case model.DeviceAccessTypeRTSP:
			if item.RtspURL != "" {
				key := "rtsp:" + item.RtspURL
				item.ExternalKey = &key
			}
		case model.DeviceAccessTypeGB28181:
			key := "gb28181:" + item.GB28181DeviceID + ":" + item.GB28181ChannelID
			item.ExternalKey = &key
		}
	}
	if req.Username != "" {
		item.Username = req.Username
	}
	// 密码留空表示不修改
	if req.Password != "" {
		hashed, err := hash.Hash(req.Password)
		if err != nil {
			zap.L().Error("hash device password failed", zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		item.Password = hashed
	}
	if req.Manufacturer != "" {
		item.Manufacturer = req.Manufacturer
	}
	if req.Model != "" {
		item.Model = req.Model
	}
	if req.FirmwareVersion != "" {
		item.FirmwareVersion = req.FirmwareVersion
	}
	if req.Latitude != nil {
		item.Latitude = req.Latitude
	}
	if req.Longitude != nil {
		item.Longitude = req.Longitude
	}
	if req.LocationDesc != "" {
		item.LocationDesc = req.LocationDesc
	}
	if req.Remark != "" {
		item.Remark = req.Remark
	}
	if req.Status != "" {
		item.Status = req.Status
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}

	if err := s.deviceRepo.Update(ctx, item); err != nil {
		zap.L().Error("update device failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	// 同步设备分组关联(显式传入空数组也视为清空)
	if req.GroupIDs != nil {
		if err := s.deviceRepo.ReplaceGroups(ctx, item.ID, req.GroupIDs); err != nil {
			zap.L().Error("sync device groups failed", zap.Error(err))
		}
	}

	// 重新加载分组信息以返回完整数据
	item, _ = s.deviceRepo.FindByID(ctx, item.ID)

	// 1. 如果状态发生变更,同步更新缓存
	if req.Status != "" {
		statusData, _ := json.Marshal(item.Status)
		_ = s.cache.Set(ctx, s.getStatusCacheKey(item.ID), statusData, 24*time.Hour)
	}

	// 2. 如果修改了关键接入参数,重新触发异步探测
	if req.AccessType != "" || req.RtspURL != "" || req.GB28181DeviceID != "" {
		_ = s.taskClient.Enqueue(ctx, "device:detect", map[string]string{"id": item.ID})
	}

	return toDeviceResponse(item), nil
}

// Delete 删除设备
func (s *DeviceService) Delete(ctx context.Context, id string) error {
	if _, err := s.deviceRepo.FindByID(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrDeviceNotFound, "")
	}

	// TODO: 校验设备是否被推理任务绑定(后续对接 InferTaskRepository)
	// if bound, return ErrBadRequest "设备已绑定任务,请先停止或解绑任务"

	if err := s.deviceRepo.Delete(ctx, id); err != nil {
		zap.L().Error("delete device failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	// 重置关联的待接入设备状态为 pending
	if err := s.discoveredDeviceRepo.ResetByDeviceID(ctx, id); err != nil {
		zap.L().Warn("reset discovered device status failed", zap.String("device_id", id), zap.Error(err))
	}

	// 关闭 ZLM 中对应的代理流
	_ = s.zlmClient.CloseStream(ctx, zlm.CloseStreamRequest{
		Vhost:  "__defaultVhost__",
		App:    "live",
		Stream: id,
		Force:  1,
	})

	return nil
}

// BatchDelete 批量删除设备
func (s *DeviceService) BatchDelete(ctx context.Context, ids []string) *dto.BatchResult {
	items, err := s.deviceRepo.FindByIDs(ctx, ids)
	if err != nil {
		zap.L().Error("batch delete find devices failed", zap.Error(err))
		return &dto.BatchResult{
			Total: len(ids), Success: 0, Failed: len(ids),
			Items: batchErrorItems(ids, apperrors.ErrInternal, ""),
		}
	}

	validIDs := make([]string, 0, len(items))
	for _, item := range items {
		validIDs = append(validIDs, item.ID)
	}

	if err := s.deviceRepo.BatchDelete(ctx, validIDs); err != nil {
		zap.L().Error("batch delete devices failed", zap.Error(err))
		return &dto.BatchResult{
			Total: len(ids), Success: 0, Failed: len(ids),
			Items: batchErrorItems(ids, apperrors.ErrInternal, ""),
		}
	}

	// 批量重置关联的待接入设备状态为 pending
	for _, id := range validIDs {
		if err := s.discoveredDeviceRepo.ResetByDeviceID(ctx, id); err != nil {
			zap.L().Warn("reset discovered device status failed", zap.String("device_id", id), zap.Error(err))
		}
	}

	return &dto.BatchResult{
		Total: len(ids), Success: len(validIDs), Failed: len(ids) - len(validIDs),
	}
}

// TestConnection 测试设备连接,进行真实的流可达性探测并更新设备状态
func (s *DeviceService) TestConnection(ctx context.Context, id string, lang string) (*dto.DeviceTestResultResponse, error) {
	_, err := s.deviceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrDeviceNotFound, "")
	}

	now := time.Now().Format(time.RFC3339)

	// 使用 StreamManager 进行连接探测(Acquire reason="detect")
	err = s.streamManager.Acquire(ctx, id, "detect", nil)

	testSuccess := err == nil
	testMessage := "测试连接成功，流可达"
	if err != nil {
		// 完整错误记入日志，不暴露内部细节给前端
		zap.L().Warn("device connection test failed",
			zap.String("device_id", id),
			zap.Error(err),
		)
		errCode := classifyConnectionError(err)
		testMessage = i18n.Translate(lang, errCode)
	}

	// 探测完成后立即释放
	defer func() {
		_ = s.streamManager.Release(ctx, id, "detect")
	}()

	// 根据测试结果更新设备状态
	newStatus := model.DeviceStatusOffline
	errorCode := ""
	errorMessage := ""
	if testSuccess {
		newStatus = model.DeviceStatusOnline
	} else {
		errorCode = "CONNECTION_TEST_FAILED"
		errorMessage = testMessage
	}

	// 更新设备状态和错误信息
	if err := s.deviceRepo.UpdateStatus(ctx, id, newStatus, errorCode, errorMessage); err != nil {
		zap.L().Error("update device status after test failed", zap.String("device_id", id), zap.Error(err))
	}

	// 更新缓存
	statusData, _ := json.Marshal(newStatus)
	_ = s.cache.Set(ctx, s.getStatusCacheKey(id), statusData, 24*time.Hour)

	return &dto.DeviceTestResultResponse{
		Success:  testSuccess,
		Message:  testMessage,
		TestedAt: now,
	}, nil
}

// classifyConnectionError 根据错误类型返回错误码,供 i18n 翻译。
func classifyConnectionError(err error) int {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "dial tcp"):
		return apperrors.ErrEngineNotReady
	case strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "deadline exceeded"):
		return apperrors.ErrConnectionTimeout
	case strings.Contains(msg, "read resp"):
		return apperrors.ErrEngineResponseBad
	default:
		return apperrors.ErrConnectionFailed
	}
}

// ExportCSV 导出设备列表为 CSV
func (s *DeviceService) ExportCSV(ctx context.Context, req dto.DeviceListRequest) ([]byte, error) {
	items, _, err := s.deviceRepo.List(ctx, req)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	headers := []string{"设备名称", "接入类型", "状态", "是否启用", "制造商", "型号", "固件版本", "创建时间"}
	rows := make([][]string, len(items))
	for i, item := range items {
		enabled := "是"
		if !item.Enabled {
			enabled = "否"
		}
		rows[i] = []string{
			item.DeviceName,
			item.AccessType,
			item.Status,
			enabled,
			item.Manufacturer,
			item.Model,
			item.FirmwareVersion,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	data, err := csvx.Build(headers, rows)
	if err != nil {
		zap.L().Error("build device csv failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return data, nil
}

// ImportCSV 从 CSV 导入设备
func (s *DeviceService) ImportCSV(ctx context.Context, reader io.Reader, lang string) (*dto.BatchResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1 // 允许可变列数

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCSVInvalidContent, "")
	}

	if len(records) < 2 { // 至少要有表头+1行数据
		return nil, apperrors.New(apperrors.ErrCSVInvalidContent, "")
	}

	// 校验表头
	header := records[0]
	expectedHeaders := []string{"设备名称", "接入类型", "RTSP URL", "国标编码", "用户名", "密码", "制造商", "备注"}
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.TrimSpace(h)] = i
	}
	for _, expected := range expectedHeaders {
		if _, ok := headerMap[expected]; !ok {
			return nil, apperrors.New(apperrors.ErrCSVHeaderInvalid, "")
		}
	}

	var result dto.BatchResult
	result.Total = len(records) - 1

	// 辅助函数:从行数据中提取指定列值
	cell := func(col string, row []string) string {
		if idx, ok := headerMap[col]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	for i := 1; i < len(records); i++ {
		row := records[i]
		lineNum := i + 1
		deviceName := cell("设备名称", row)
		accessType := cell("接入类型", row)

		if deviceName == "" {
			result.Failed++
			result.Items = append(result.Items, dto.BatchItemResult{
				Message: fmt.Sprintf("第 %d 行: 设备名称为空", lineNum),
				Success: false, Code: apperrors.ErrBadRequest,
			})
			continue
		}

		exists, err := s.deviceRepo.ExistsByName(ctx, deviceName, "")
		if err != nil {
			result.Failed++
			result.Items = append(result.Items, dto.BatchItemResult{
				Message: fmt.Sprintf("第 %d 行: 校验失败", lineNum),
				Success: false, Code: apperrors.ErrInternal,
			})
			continue
		}
		if exists {
			result.Failed++
			result.Items = append(result.Items, dto.BatchItemResult{
				Message: fmt.Sprintf("第 %d 行: 设备名称已存在", lineNum),
				Success: false, Code: apperrors.ErrBadRequest,
			})
			continue
		}

		rtspURL := cell("RTSP URL", row)
		gb28181 := cell("国标编码", row)
		username := cell("用户名", row)
		password := cell("密码", row)
		manufacturer := cell("制造商", row)
		remark := cell("备注", row)

		createReq := dto.DeviceCreateRequest{
			DeviceName:      deviceName,
			AccessType:      accessType,
			RtspURL:         rtspURL,
			GB28181DeviceID: gb28181,
			Username:        username,
			Manufacturer:    manufacturer,
			Remark:          remark,
		}

		if password != "" {
			hashed, err := hash.Hash(password)
			if err != nil {
				result.Failed++
				result.Items = append(result.Items, dto.BatchItemResult{
					Message: fmt.Sprintf("第 %d 行: 密码加密失败", lineNum),
					Success: false, Code: apperrors.ErrInternal,
				})
				continue
			}
			createReq.Password = hashed
		}

		item := &model.Device{
			DeviceName:      createReq.DeviceName,
			AccessType:      createReq.AccessType,
			RtspURL:         createReq.RtspURL,
			GB28181DeviceID: createReq.GB28181DeviceID,
			Username:        createReq.Username,
			Password:        createReq.Password,
			Manufacturer:    createReq.Manufacturer,
			Status:          model.DeviceStatusUnknown,
			Enabled:         true,
			Remark:          createReq.Remark,
		}

		// 设置 ExternalKey 用于唯一约束去重
		switch item.AccessType {
		case model.DeviceAccessTypeRTSP:
			if item.RtspURL != "" {
				key := "rtsp:" + item.RtspURL
				item.ExternalKey = &key
			}
		case model.DeviceAccessTypeGB28181:
			if item.GB28181DeviceID != "" {
				key := "gb28181:" + item.GB28181DeviceID + ":" + item.GB28181ChannelID
				item.ExternalKey = &key
			}
		}

		if err := s.deviceRepo.Create(ctx, item); err != nil {
			zap.L().Error("import device failed", zap.String("name", deviceName), zap.Error(err))
			result.Failed++
			result.Items = append(result.Items, dto.BatchItemResult{
				Message: fmt.Sprintf("第 %d 行: 导入失败", lineNum),
				Success: false, Code: apperrors.ErrInternal,
			})
			continue
		}

		result.Success++
		result.Items = append(result.Items, dto.BatchItemResult{
			ID: item.ID, Success: true,
		})
	}

	return &result, nil
}

// DeviceGroupService 处理设备分组管理的业务逻辑
type DeviceGroupService struct {
	groupRepo deviceGroupRepo
}

// NewDeviceGroupService 创建并返回一个新的 DeviceGroupService 实例
func NewDeviceGroupService(repo deviceGroupRepo) *DeviceGroupService {
	return &DeviceGroupService{groupRepo: repo}
}

// List 返回分页的设备分组列表(含设备数量)
func (s *DeviceGroupService) List(ctx context.Context, req dto.DeviceGroupListRequest) ([]dto.DeviceGroupResponse, int64, error) {
	items, total, err := s.groupRepo.List(ctx, req)
	if err != nil {
		zap.L().Error("device group list failed", zap.Error(err))
		return nil, 0, apperrors.New(apperrors.ErrInternal, "")
	}

	result := make([]dto.DeviceGroupResponse, len(items))
	for i, item := range items {
		count, _ := s.groupRepo.CountByGroupID(ctx, item.ID)
		resp := toDeviceGroupResponse(&item, count)
		result[i] = *resp
	}
	return result, total, nil
}

// GetByID 根据 ID 获取设备分组详情
func (s *DeviceGroupService) GetByID(ctx context.Context, id string) (*dto.DeviceGroupResponse, error) {
	item, err := s.groupRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrDeviceGroupNotFound, "")
	}
	count, _ := s.groupRepo.CountByGroupID(ctx, item.ID)
	return toDeviceGroupResponse(item, count), nil
}

// Create 创建设备分组
func (s *DeviceGroupService) Create(ctx context.Context, req dto.DeviceGroupCreateRequest) (*dto.DeviceGroupResponse, error) {
	item := &model.DeviceGroup{
		GroupName:   req.GroupName,
		Description: req.Description,
		ParentID:    req.ParentID,
		SortOrder:   req.SortOrder,
	}
	if err := s.groupRepo.Create(ctx, item); err != nil {
		zap.L().Error("create device group failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return toDeviceGroupResponse(item, 0), nil
}

// Update 更新设备分组
func (s *DeviceGroupService) Update(ctx context.Context, id string, req dto.DeviceGroupUpdateRequest) (*dto.DeviceGroupResponse, error) {
	item, err := s.groupRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrDeviceGroupNotFound, "")
	}

	if req.GroupName != "" {
		item.GroupName = req.GroupName
	}
	if req.Description != "" {
		item.Description = req.Description
	}
	if req.ParentID != nil {
		item.ParentID = req.ParentID
	}
	if req.SortOrder != 0 {
		item.SortOrder = req.SortOrder
	}

	if err := s.groupRepo.Update(ctx, item); err != nil {
		zap.L().Error("update device group failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	count, _ := s.groupRepo.CountByGroupID(ctx, item.ID)
	return toDeviceGroupResponse(item, count), nil
}

// Delete 删除设备分组
func (s *DeviceGroupService) Delete(ctx context.Context, id string) error {
	if _, err := s.groupRepo.FindByID(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrDeviceGroupNotFound, "")
	}

	count, err := s.groupRepo.CountByGroupID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	if count > 0 {
		return apperrors.New(apperrors.ErrDeviceGroupNotEmpty, "")
	}

	if err := s.groupRepo.Delete(ctx, id); err != nil {
		zap.L().Error("delete device group failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}
	return nil
}

// ---------- 辅助函数 ----------

func toDeviceListResponse(item *model.Device) dto.DeviceListResponse {
	resp := dto.DeviceListResponse{
		ID:           item.ID,
		DeviceName:   item.DeviceName,
		AccessType:   item.AccessType,
		Status:       item.Status,
		Enabled:      item.Enabled,
		Manufacturer: item.Manufacturer,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
	if item.LastOnlineAt != nil {
		s := item.LastOnlineAt.Format(time.RFC3339)
		resp.LastOnlineAt = &s
	}
	if item.LastOfflineAt != nil {
		s := item.LastOfflineAt.Format(time.RFC3339)
		resp.LastOfflineAt = &s
	}
	return resp
}

func toDeviceResponse(item *model.Device) *dto.DeviceResponse {
	resp := &dto.DeviceResponse{
		ID:               item.ID,
		DeviceName:       item.DeviceName,
		AccessType:       item.AccessType,
		RtspURL:          maskRtspPassword(item.RtspURL),
		GB28181DeviceID:  item.GB28181DeviceID,
		GB28181ChannelID: item.GB28181ChannelID,
		Username:         item.Username,
		Manufacturer:     item.Manufacturer,
		Model:            item.Model,
		FirmwareVersion:  item.FirmwareVersion,
		Status:           item.Status,
		Enabled:          item.Enabled,
		Latitude:         item.Latitude,
		Longitude:        item.Longitude,
		LocationDesc:     item.LocationDesc,
		LastErrorCode:    item.LastErrorCode,
		LastErrorMessage: item.LastErrorMessage,
		Remark:           item.Remark,
		Version:          item.Version,
		CreatedBy:        item.CreatedBy,
	}

	if item.ExternalKey != nil {
		resp.ExternalKey = *item.ExternalKey
	}

	if item.LastOnlineAt != nil {
		s := item.LastOnlineAt.Format(time.RFC3339)
		resp.LastOnlineAt = &s
	}
	if item.LastOfflineAt != nil {
		s := item.LastOfflineAt.Format(time.RFC3339)
		resp.LastOfflineAt = &s
	}
	if !item.CreatedAt.IsZero() {
		resp.CreatedAt = item.CreatedAt.Format(time.RFC3339)
	}
	if !item.UpdatedAt.IsZero() {
		resp.UpdatedAt = item.UpdatedAt.Format(time.RFC3339)
	}
	if len(item.Groups) > 0 {
		resp.Groups = make([]dto.DeviceGroupResponse, len(item.Groups))
		for i, g := range item.Groups {
			resp.Groups[i] = *toDeviceGroupResponse(&g, 0)
		}
	}
	return resp
}

func toDeviceGroupResponse(item *model.DeviceGroup, deviceCount int64) *dto.DeviceGroupResponse {
	return &dto.DeviceGroupResponse{
		ID:          item.ID,
		GroupName:   item.GroupName,
		Description: item.Description,
		SortOrder:   item.SortOrder,
		DeviceCount: deviceCount,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}
}

// maskRtspPassword 脱敏 RTSP URL 中的密码段
func maskRtspPassword(url string) string {
	if url == "" {
		return ""
	}
	// rtsp://user:password@host/path → rtsp://user:****@host/path
	afterProtocol := strings.TrimPrefix(url, "rtsp://")
	if afterProtocol == url {
		return url
	}
	atIndex := strings.Index(afterProtocol, "@")
	if atIndex == -1 {
		return url
	}
	userInfo := afterProtocol[:atIndex]
	colonIndex := strings.Index(userInfo, ":")
	if colonIndex == -1 {
		return url
	}
	return "rtsp://" + userInfo[:colonIndex+1] + "****" + afterProtocol[atIndex:]
}

func batchErrorItems(ids []string, code int, message string) []dto.BatchItemResult {
	items := make([]dto.BatchItemResult, len(ids))
	for i, id := range ids {
		items[i] = dto.BatchItemResult{ID: id, Success: false, Code: code, Message: message}
	}
	return items
}
