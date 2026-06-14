package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// SmartRecordRepository handles SmartRecord persistence.
type SmartRecordRepository struct {
	db *gorm.DB
}

// NewSmartRecordRepository creates a new SmartRecordRepository.
func NewSmartRecordRepository(db *gorm.DB) *SmartRecordRepository {
	return &SmartRecordRepository{db: db}
}

// Create inserts a new smart record.
func (r *SmartRecordRepository) Create(ctx context.Context, record *model.SmartRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// CreateInBatches inserts smart records in batches.
func (r *SmartRecordRepository) CreateInBatches(ctx context.Context, records []model.SmartRecord, batchSize int) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(records, batchSize).Error
}

// FindByID finds a record by its ID.
func (r *SmartRecordRepository) FindByID(ctx context.Context, id string) (*model.SmartRecord, error) {
	var record model.SmartRecord
	err := r.db.WithContext(ctx).First(&record, "record_id = ?", id).Error
	return &record, err
}

// baseQuery 构建智能记录的基础查询：应用筛选条件、关联表 JOIN 和额外过滤。
// 智能记录使用复合主键 (record_id, capture_time)，没有 id 列，
// 因此不能使用 scopes.OrderByDefault()（会生成 "ORDER BY id DESC" 报错）。
// 调用方需自行添加排序、分页或 LIMIT。
func (r *SmartRecordRepository) baseQuery(ctx context.Context, req dto.SmartRecordListRequest) *gorm.DB {
	query := r.db.WithContext(ctx).Table("smart_records").Scopes(req.FilterScopes()...)
	query = r.joinRelations(query)
	query = r.applyExtraFilters(query, req)
	return query
}

// applyRecordSort 显式双字段排序：capture_time DESC 为默认主排序，created_at DESC 作为稳定 tiebreaker。
func applyRecordSort(query *gorm.DB, req dto.SmartRecordListRequest) *gorm.DB {
	return query.Scopes(scopes.OrderBy(req.Sort, req.Order, model.SmartRecord{}.SortableFields()...)).
		Order("smart_records.capture_time DESC").
		Order("smart_records.created_at DESC")
}

// List 分页查询智能记录。
func (r *SmartRecordRepository) List(ctx context.Context, req dto.SmartRecordListRequest) ([]model.SmartRecord, int64, error) {
	var items []model.SmartRecord
	var total int64

	query := r.baseQuery(ctx, req)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := applyRecordSort(query, req).
		Scopes(scopes.Paginate(req.GetPage(), req.GetPageSize())).
		Find(&items).Error
	return items, total, err
}

// ListForExport 返回符合筛选条件的智能记录，用于导出。
func (r *SmartRecordRepository) ListForExport(ctx context.Context, req dto.SmartRecordListRequest, limit int) ([]model.SmartRecord, error) {
	var items []model.SmartRecord

	err := applyRecordSort(r.baseQuery(ctx, req), req).
		Limit(limit).
		Find(&items).Error
	return items, err
}

// BatchDelete 批量删除智能记录（硬删除，因为 smart_records 是分区表无软删除）。
func (r *SmartRecordRepository) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("record_id IN ?", ids).Delete(&model.SmartRecord{}).Error
}

// UpdateAlarmStatus 更新告警记录处理状态。
func (r *SmartRecordRepository) UpdateAlarmStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.SmartRecord{}).
		Where("record_id = ? AND record_type = ?", id, model.RecordTypeAlarm).
		Update("alarm_status", status).Error
}

// FindByIDs 根据 ID 列表查询智能记录（含底库图 JOIN）。
func (r *SmartRecordRepository) FindByIDs(ctx context.Context, ids []string) ([]model.SmartRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []model.SmartRecord
	query := r.db.WithContext(ctx).Table("smart_records").Where("smart_records.record_id IN ?", ids)
	query = r.joinRelations(query)
	err := query.Find(&items).Error
	return items, err
}

// ListCategoryCodes 返回所有启用的类别编码，用于前端下拉选择。
func (r *SmartRecordRepository) ListCategoryCodes(ctx context.Context) ([]model.CategoryCode, error) {
	var items []model.CategoryCode
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("category_code ASC").Find(&items).Error
	return items, err
}

// joinRelations 将 persons 和 category_codes 表 LEFT JOIN 到查询中，用于获取底库人脸图 URL 和类别显示名称。
// 注意：两张关联表均有索引（persons.id, category_codes.category_code），当前性能可接受。
// 若 smart_records 数据量增长至千万级，应考虑预计算或缓存。
func (r *SmartRecordRepository) joinRelations(query *gorm.DB) *gorm.DB {
	return query.
		Select(`smart_records.*,
			COALESCE(p.image_url, '') AS person_image_url,
			COALESCE(cc.display_name, '') AS category_name`).
		Joins("LEFT JOIN persons p ON p.id = smart_records.person_record_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN category_codes cc ON cc.category_code = smart_records.category_code")
}

func (r *SmartRecordRepository) applyExtraFilters(query *gorm.DB, req dto.SmartRecordListRequest) *gorm.DB {
	if req.MinConfidence != nil {
		query = query.Where("confidence >= ?", *req.MinConfidence)
	}
	if req.MaxConfidence != nil {
		query = query.Where("confidence <= ?", *req.MaxConfidence)
	}
	if req.MinSimilarity != nil {
		query = query.Where("similarity >= ?", *req.MinSimilarity)
	}
	if req.MaxSimilarity != nil {
		query = query.Where("similarity <= ?", *req.MaxSimilarity)
	}
	if req.BusinessTag != "" {
		query = query.Where("? = ANY(business_tags)", req.BusinessTag)
	}
	if req.GroupID != "" {
		query = query.Where(
			"EXISTS (SELECT 1 FROM device_group_members dgm WHERE dgm.device_id = smart_records.device_id AND dgm.group_id = ?)",
			req.GroupID,
		)
	}
	return query
}
