package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/repository"
)

type SmartRecordHandler struct {
	repo *repository.SmartRecordRepository
}

func NewSmartRecordHandler(repo *repository.SmartRecordRepository) *SmartRecordHandler {
	return &SmartRecordHandler{repo: repo}
}

// List 智能记录列表
// @Summary      智能记录列表
// @Tags         智能记录
// @Produce      json
// @Param        page        query   int     false  "页码"
// @Param        page_size   query   int     false  "每页数量"
// @Param        type        query   string  false  "记录类型 (如 alarm)"
// @Param        device_id   query   string  false  "设备 ID"
// @Param        alarm_type  query   string  false  "告警类型"
// @Param        alarm_level query   string  false  "告警级别"
// @Param        start_time  query   string  false  "开始时间"
// @Param        end_time    query   string  false  "结束时间"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]dto.SmartRecordResponse}}
// @Router       /smart-records [get]
// @Security     BearerAuth
func (h *SmartRecordHandler) List(c *gin.Context) {
	var req dto.SmartRecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	filters := buildSmartRecordFilters(req)

	items, total, err := h.repo.List(c.Request.Context(), filters, req.GetPage(), req.GetPageSize())
	if err != nil {
		attachError(c, err)
		return
	}

	var list []dto.SmartRecordResponse
	for _, item := range items {
		list = append(list, smartRecordToResponse(item))
	}

	response.Page(c, list, total, req.GetPage(), req.GetPageSize())
}

// ExportCSV 导出智能记录 CSV
// @Summary      导出智能记录
// @Tags         智能记录
// @Produce      text/csv
// @Param        type        query   string  false  "记录类型 (如 alarm)"
// @Param        device_id   query   string  false  "设备 ID"
// @Param        alarm_type  query   string  false  "告警类型"
// @Param        alarm_level query   string  false  "告警级别"
// @Param        start_time  query   string  false  "开始时间"
// @Param        end_time    query   string  false  "结束时间"
// @Success      200  {file}  file
// @Router       /smart-records/export [get]
// @Security     BearerAuth
func (h *SmartRecordHandler) ExportCSV(c *gin.Context) {
	var req dto.SmartRecordListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}

	filters := buildSmartRecordFilters(req)

	items, _, err := h.repo.List(c.Request.Context(), filters, 1, 10000)
	if err != nil {
		attachError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=smart_records.csv")
	c.Header("Content-Type", "text/csv")
	c.Status(http.StatusOK)

	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"RecordID", "Type", "DeviceID", "AlarmType", "AlarmLevel", "CreatedAt", "RawResult"})

	for _, item := range items {
		_ = writer.Write([]string{
			item.RecordID,
			item.RecordType,
			safeDeref(item.DeviceID),
			item.AlarmType,
			item.AlarmLevel,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
			string(item.RawResult),
		})
	}
	writer.Flush()
}

// safeDeref 安全解引用 string 指针，nil 返回空字符串。
func safeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// smartRecordToResponse 将 model.SmartRecord 转换为 dto.SmartRecordResponse。
func smartRecordToResponse(item model.SmartRecord) dto.SmartRecordResponse {
	var rawResult interface{}
	_ = json.Unmarshal(item.RawResult, &rawResult)
	return dto.SmartRecordResponse{
		RecordID:         item.RecordID,
		RecordType:       item.RecordType,
		DeviceID:         safeDeref(item.DeviceID),
		DeviceName:       item.DeviceName,
		TaskName:         item.TaskName,
		AlarmType:        item.AlarmType,
		AlarmLevel:       item.AlarmLevel,
		SnapshotImageURL: item.SnapshotImageURL,
		Confidence:       item.Confidence,
		RawResult:        rawResult,
		CreatedAt:        item.CreatedAt,
	}
}

// buildSmartRecordFilters 构建 SmartRecord 查询过滤器，自动跳过空值。
func buildSmartRecordFilters(req dto.SmartRecordListRequest) map[string]interface{} {
	filters := map[string]interface{}{}
	if req.RecordType != "" {
		filters["record_type"] = req.RecordType
	}
	if req.DeviceID != "" {
		filters["device_id"] = req.DeviceID
	}
	if req.AlarmType != "" {
		filters["alarm_type"] = req.AlarmType
	}
	if req.AlarmLevel != "" {
		filters["alarm_level"] = req.AlarmLevel
	}
	if req.StartTime != "" {
		filters["start_time"] = req.StartTime
	}
	if req.EndTime != "" {
		filters["end_time"] = req.EndTime
	}
	return filters
}
