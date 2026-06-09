package service

import (
	"context"
	"strconv"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/csvx"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/timex"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// SmartRecordService 处理智能记录查询与导出。
type SmartRecordService struct {
	repo *repository.SmartRecordRepository
}

// NewSmartRecordService 创建智能记录服务。
func NewSmartRecordService(repo *repository.SmartRecordRepository) *SmartRecordService {
	return &SmartRecordService{repo: repo}
}

// ListCategoryCodes 返回所有启用的类别编码供前端下拉使用。
func (s *SmartRecordService) ListCategoryCodes(ctx context.Context) ([]dto.CategoryCodeOption, error) {
	items, err := s.repo.ListCategoryCodes(ctx)
	if err != nil {
		return nil, err
	}
	opts := make([]dto.CategoryCodeOption, 0, len(items))
	for _, item := range items {
		label := item.DisplayName
		if label == "" {
			label = strconv.Itoa(item.CategoryCode)
		}
		opts = append(opts, dto.CategoryCodeOption{Value: item.CategoryCode, Label: label})
	}
	return opts, nil
}

// List 查询智能记录列表。
func (s *SmartRecordService) List(ctx context.Context, req dto.SmartRecordListRequest) ([]model.SmartRecord, int64, error) {
	if err := normalizeSmartRecordTimeRange(&req); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, req)
}

// BatchDelete 批量删除智能记录。
func (s *SmartRecordService) BatchDelete(ctx context.Context, ids []string) *dto.BatchResult {
	if len(ids) == 0 {
		return &dto.BatchResult{Total: 0, Success: 0, Failed: 0}
	}
	if err := s.repo.BatchDelete(ctx, ids); err != nil {
		zap.L().Error("batch delete smart records failed", zap.Error(err))
		return &dto.BatchResult{
			Total:   len(ids),
			Success: 0,
			Failed:  len(ids),
		}
	}
	return &dto.BatchResult{
		Total:   len(ids),
		Success: len(ids),
		Failed:  0,
	}
}

// UpdateAlarmStatus 更新告警记录处理状态。
func (s *SmartRecordService) UpdateAlarmStatus(ctx context.Context, id string, status string) error {
	if status != "unhandled" && status != "handled" {
		return apperrors.New(apperrors.ErrBadRequest, "")
	}
	if err := s.repo.UpdateAlarmStatus(ctx, id, status); err != nil {
		zap.L().Error("update smart record alarm status failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}
	return nil
}

// ExportSelectedCSV 按选定 ID 列表导出智能记录 CSV。
func (s *SmartRecordService) ExportSelectedCSV(ctx context.Context, ids []string) ([]byte, error) {
	if len(ids) == 0 {
		return csvx.Build(csvHeaders, nil)
	}
	if len(ids) > maxExportSelected {
		return nil, apperrors.New(apperrors.ErrBadRequest, "")
	}
	items, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		zap.L().Error("export selected smart records failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return buildSmartRecordCSV(items)
}

// ExportCSV 按当前筛选条件导出智能记录 CSV。
func (s *SmartRecordService) ExportCSV(ctx context.Context, req dto.SmartRecordListRequest) ([]byte, error) {
	if err := normalizeSmartRecordTimeRange(&req); err != nil {
		return nil, err
	}
	items, err := s.repo.ListForExport(ctx, req, maxCSVExportRows)
	if err != nil {
		zap.L().Error("export smart records failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	return buildSmartRecordCSV(items)
}

// csvHeaders 智能记录 CSV 列头定义。
var csvHeaders = []string{"RecordID", "Type", "Device", "Task", "Algorithm", "CaptureTime", "Person", "PersonImage", "Similarity", "AlarmType", "AlarmLevel", "Category", "Confidence"}

// buildSmartRecordCSV 将智能记录列表构建为 CSV 字节数组。
func buildSmartRecordCSV(items []model.SmartRecord) ([]byte, error) {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		category := item.CategoryName
		if category == "" {
			category = formatIntPtr(item.CategoryCode)
		}
		rows = append(rows, []string{
			item.RecordID,
			item.RecordType,
			item.DeviceName,
			item.TaskName,
			item.AlgorithmName,
			item.CaptureTime.Format("2006-01-02 15:04:05"),
			item.PersonName,
			item.PersonImageURL,
			formatFloatPtr(item.Similarity),
			item.AlarmType,
			item.AlarmLevel,
			category,
			formatFloatPtr(item.Confidence),
		})
	}
	return csvx.Build(csvHeaders, rows)
}

func normalizeSmartRecordTimeRange(req *dto.SmartRecordListRequest) error {
	return timex.NormalizeRange(req.StartTime, req.EndTime, &req.FromTime, &req.ToTime)
}

func formatFloatPtr(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', 4, 64)
}

func formatIntPtr(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}
