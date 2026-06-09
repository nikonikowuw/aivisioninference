// Package repository 提供人员管理模块的数据访问层。
package repository

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
	"gorm.io/gorm"
)

// PersonRepository 处理人员记录持久化。
type PersonRepository struct{ db *gorm.DB }

// NewPersonRepository 创建人员 Repository。
func NewPersonRepository(db *gorm.DB) *PersonRepository { return &PersonRepository{db: db} }

// DB 返回底层 DB，供事务型服务复用。
func (r *PersonRepository) DB() *gorm.DB { return r.db }

// Create 创建人员记录。
func (r *PersonRepository) Create(ctx context.Context, item *model.Person) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByID 根据 ID 查询人员。
func (r *PersonRepository) FindByID(ctx context.Context, id string) (*model.Person, error) {
	var item model.Person
	err := r.db.WithContext(ctx).Preload("Groups").Preload("Embedding").First(&item, "id = ?", id).Error
	return &item, err
}

// FindByIDs 根据 ID 列表查询人员。
func (r *PersonRepository) FindByIDs(ctx context.Context, ids []string) ([]model.Person, error) {
	var items []model.Person
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&items).Error
	return items, err
}

// List 查询人员分页列表。
func (r *PersonRepository) List(ctx context.Context, req dto.PersonListRequest) ([]model.Person, int64, error) {
	base := r.db.WithContext(ctx).Model(&model.Person{}).Scopes(req.FilterScopes()...)
	if req.GroupID != "" {
		base = base.Joins("JOIN person_group_members pgm ON pgm.person_record_id = persons.id AND pgm.group_id = ?", req.GroupID)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Person
	err := base.Scopes(scopes.OrderBy(req.Sort, req.Order, model.Person{}.SortableFields()...), scopes.OrderByDefault(), scopes.Paginate(req.GetPage(), req.GetPageSize())).
		Preload("Groups").Find(&items).Error
	return items, total, err
}

// ListForExport 查询导出列表。
func (r *PersonRepository) ListForExport(ctx context.Context, req dto.PersonListRequest, limit int) ([]model.Person, error) {
	var items []model.Person
	q := r.db.WithContext(ctx).Model(&model.Person{}).Scopes(req.FilterScopes()...)
	if req.GroupID != "" {
		q = q.Joins("JOIN person_group_members pgm ON pgm.person_record_id = persons.id AND pgm.group_id = ?", req.GroupID)
	}
	return items, q.Scopes(scopes.OrderBy(req.Sort, req.Order, model.Person{}.SortableFields()...), scopes.OrderByDefault()).Limit(limit).Preload("Groups").Find(&items).Error
}

// Update 更新人员记录，仅更新业务字段（不覆盖零值）。
func (r *PersonRepository) Update(ctx context.Context, item *model.Person) error {
	return r.db.WithContext(ctx).Model(item).Updates(map[string]interface{}{
		"person_code":                 item.PersonCode,
		"person_name":                 item.PersonName,
		"gender":                      item.Gender,
		"phone":                       item.Phone,
		"id_number":                   item.IDNumber,
		"image_url":                   item.ImageURL,
		"image_md5":                   item.ImageMD5,
		"face_quality_score":          item.FaceQualityScore,
		"embedding_status":            item.EmbeddingStatus,
		"embedding_error_code":        item.EmbeddingErrorCode,
		"embedding_error_message_key": item.EmbeddingErrorMessageKey,
		"embedding_retryable":         item.EmbeddingRetryable,
		"enabled":                     item.Enabled,
		"remark":                      item.Remark,
	}).Error
}

// SoftDelete 软删除人员。
func (r *PersonRepository) SoftDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Person{}, "id = ?", id).Error
}

// BatchSoftDelete 批量软删除人员。
func (r *PersonRepository) BatchSoftDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Delete(&model.Person{}, "id IN ?", ids).Error
}

// BatchToggle 批量启用/禁用。
func (r *PersonRepository) BatchToggle(ctx context.Context, ids []string, enabled bool) error {
	return r.db.WithContext(ctx).Model(&model.Person{}).Where("id IN ?", ids).Update("enabled", enabled).Error
}

// UpdateEmbeddingStatus 更新特征状态和错误信息。
func (r *PersonRepository) UpdateEmbeddingStatus(ctx context.Context, id, status, code, messageKey string, retryable bool) error {
	return r.db.WithContext(ctx).Model(&model.Person{}).Where("id = ?", id).Updates(map[string]interface{}{
		"embedding_status":            status,
		"embedding_error_code":        code,
		"embedding_error_message_key": messageKey,
		"embedding_retryable":         retryable,
	}).Error
}

// ExistsByPersonCode 检查人员编号是否存在。
func (r *PersonRepository) ExistsByPersonCode(ctx context.Context, code, excludeID string) (bool, error) {
	if code == "" { return false, nil }
	q := r.db.WithContext(ctx).Model(&model.Person{}).Where("person_code = ?", code)
	if excludeID != "" { q = q.Where("id <> ?", excludeID) }
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

// ExistsByImageMD5 检查图片 MD5 是否存在。
func (r *PersonRepository) ExistsByImageMD5(ctx context.Context, md5, excludeID string) (bool, error) {
	if md5 == "" { return false, nil }
	q := r.db.WithContext(ctx).Model(&model.Person{}).Where("image_md5 = ?", md5)
	if excludeID != "" { q = q.Where("id <> ?", excludeID) }
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

// ReplaceGroups 替换人员分组关联。
func (r *PersonRepository) ReplaceGroups(ctx context.Context, personID string, groupIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("person_record_id = ?", personID).Delete(&model.PersonGroupMember{}).Error; err != nil {
			return err
		}
		if len(groupIDs) == 0 {
			return nil
		}
		members := make([]model.PersonGroupMember, 0, len(groupIDs))
		for _, id := range groupIDs {
			if id == "" {
				continue
			}
			members = append(members, model.PersonGroupMember{PersonRecordID: personID, GroupID: id})
		}
		if len(members) > 0 {
			return tx.Create(&members).Error
		}
		return nil
	})
}

// PersonGroupRepository 处理人员分组持久化。
type PersonGroupRepository struct{ db *gorm.DB }

// NewPersonGroupRepository 创建人员分组 Repository。
func NewPersonGroupRepository(db *gorm.DB) *PersonGroupRepository { return &PersonGroupRepository{db: db} }

func (r *PersonGroupRepository) Create(ctx context.Context, item *model.PersonGroup) error { return r.db.WithContext(ctx).Create(item).Error }
func (r *PersonGroupRepository) FindByID(ctx context.Context, id string) (*model.PersonGroup, error) { var item model.PersonGroup; err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; return &item, err }
func (r *PersonGroupRepository) Update(ctx context.Context, item *model.PersonGroup) error { return r.db.WithContext(ctx).Save(item).Error }
func (r *PersonGroupRepository) Delete(ctx context.Context, id string) error { return r.db.WithContext(ctx).Delete(&model.PersonGroup{}, "id = ?", id).Error }
func (r *PersonGroupRepository) List(ctx context.Context) ([]model.PersonGroup, error) { var items []model.PersonGroup; err := r.db.WithContext(ctx).Order("sort_order ASC").Order("created_at DESC").Find(&items).Error; return items, err }
func (r *PersonGroupRepository) CountPersons(ctx context.Context, id string) (int64, error) { var count int64; err := r.db.WithContext(ctx).Model(&model.PersonGroupMember{}).Where("group_id = ?", id).Count(&count).Error; return count, err }
func (r *PersonGroupRepository) BatchCountPersons(ctx context.Context) (map[string]int64, error) {
	counts := make(map[string]int64)
	var rows []struct {
		GroupID string `gorm:"column:group_id"`
		Count   int64  `gorm:"column:count"`
	}
	err := r.db.WithContext(ctx).Model(&model.PersonGroupMember{}).Select("group_id, COUNT(*) as count").Group("group_id").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.GroupID] = row.Count
	}
	return counts, nil
}

// PersonEmbeddingRepository 处理人员向量持久化。
type PersonEmbeddingRepository struct{ db *gorm.DB }

// NewPersonEmbeddingRepository 创建人员向量 Repository。
func NewPersonEmbeddingRepository(db *gorm.DB) *PersonEmbeddingRepository { return &PersonEmbeddingRepository{db: db} }

func (r *PersonEmbeddingRepository) DeleteByPersonRecordID(ctx context.Context, personID string) error { return r.db.WithContext(ctx).Where("person_record_id = ?", personID).Delete(&model.PersonEmbedding{}).Error }
func (r *PersonEmbeddingRepository) Create(ctx context.Context, item *model.PersonEmbedding) error { return r.db.WithContext(ctx).Create(item).Error }
func (r *PersonEmbeddingRepository) ListActive(ctx context.Context, limit int) ([]model.PersonEmbedding, error) { var items []model.PersonEmbedding; if limit <= 0 || limit > 1000 { limit = 500 }; err := r.db.WithContext(ctx).Preload("Person").Limit(limit).Find(&items).Error; return items, err }

// ImportTaskRepository 处理导入任务持久化。
type ImportTaskRepository struct{ db *gorm.DB }

// NewImportTaskRepository 创建导入任务 Repository。
func NewImportTaskRepository(db *gorm.DB) *ImportTaskRepository { return &ImportTaskRepository{db: db} }

func (r *ImportTaskRepository) Create(ctx context.Context, item *model.ImportTask) error { return r.db.WithContext(ctx).Create(item).Error }
func (r *ImportTaskRepository) FindByID(ctx context.Context, id string) (*model.ImportTask, error) { var item model.ImportTask; err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; return &item, err }
func (r *ImportTaskRepository) Update(ctx context.Context, item *model.ImportTask) error {
	return r.db.WithContext(ctx).Model(item).Updates(map[string]interface{}{
		"task_type":       item.TaskType,
		"file_name":       item.FileName,
		"file_url":        item.FileURL,
		"total_rows":      item.TotalRows,
		"success_rows":    item.SuccessRows,
		"failed_rows":     item.FailedRows,
		"fail_detail_url": item.FailDetailURL,
		"status":          item.Status,
		"completed_at":    item.CompletedAt,
	}).Error
}
func (r *ImportTaskRepository) List(ctx context.Context, req dto.PageRequest) ([]model.ImportTask, int64, error) { var total int64; q := r.db.WithContext(ctx).Model(&model.ImportTask{}).Where("task_type = ?", model.ImportTaskTypePerson); if err := q.Count(&total).Error; err != nil { return nil, 0, err }; var items []model.ImportTask; err := q.Scopes(scopes.OrderBy(req.Sort, req.Order, model.ImportTask{}.SortableFields()...), scopes.OrderByDefault(), scopes.Paginate(req.GetPage(), req.GetPageSize())).Find(&items).Error; return items, total, err }

// PersonTagRepository 处理人员标签持久化。
type PersonTagRepository struct{ db *gorm.DB }

// NewPersonTagRepository 创建人员标签 Repository。
func NewPersonTagRepository(db *gorm.DB) *PersonTagRepository { return &PersonTagRepository{db: db} }

// Create 创建标签。
func (r *PersonTagRepository) Create(ctx context.Context, item *model.PersonTag) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByID 根据 ID 查询标签。
func (r *PersonTagRepository) FindByID(ctx context.Context, id string) (*model.PersonTag, error) {
	var item model.PersonTag
	err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error
	return &item, err
}

// Update 更新标签。
func (r *PersonTagRepository) Update(ctx context.Context, item *model.PersonTag) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// Delete 删除标签。
func (r *PersonTagRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.PersonTag{}, "id = ?", id).Error
}

// List 查询标签列表。
func (r *PersonTagRepository) List(ctx context.Context) ([]model.PersonTag, error) {
	var items []model.PersonTag
	err := r.db.WithContext(ctx).Order("sort_order ASC").Order("created_at DESC").Find(&items).Error
	return items, err
}

// BatchCountPersons 批量查询每个标签关联的人员数量。
func (r *PersonTagRepository) BatchCountPersons(ctx context.Context) (map[string]int64, error) {
	counts := make(map[string]int64)
	var rows []struct {
		TagID string `gorm:"column:tag_id"`
		Count int64  `gorm:"column:count"`
	}
	err := r.db.WithContext(ctx).Model(&model.PersonTagRelation{}).Select("tag_id, COUNT(*) as count").Group("tag_id").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.TagID] = row.Count
	}
	return counts, nil
}

// CountPersons 查询单个标签关联的人员数量。
func (r *PersonTagRepository) CountPersons(ctx context.Context, id string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.PersonTagRelation{}).Where("tag_id = ?", id).Count(&count).Error
	return count, err
}

// PersonTagRelationRepository 处理人员与标签关联持久化。
type PersonTagRelationRepository struct{ db *gorm.DB }

// NewPersonTagRelationRepository 创建人员标签关联 Repository。
func NewPersonTagRelationRepository(db *gorm.DB) *PersonTagRelationRepository { return &PersonTagRelationRepository{db: db} }

// BatchCreate 批量创建人员标签关联。
func (r *PersonTagRelationRepository) BatchCreate(ctx context.Context, personID string, tagIDs []string) error {
	if len(tagIDs) == 0 {
		return nil
	}
	relations := make([]model.PersonTagRelation, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		if tagID == "" {
			continue
		}
		relations = append(relations, model.PersonTagRelation{PersonRecordID: personID, TagID: tagID})
	}
	if len(relations) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&relations).Error
}

// DeleteByPersonRecordID 删除人员的所有标签关联。
func (r *PersonTagRelationRepository) DeleteByPersonRecordID(ctx context.Context, personID string) error {
	return r.db.WithContext(ctx).Where("person_record_id = ?", personID).Delete(&model.PersonTagRelation{}).Error
}

// ListTagIDsByPerson 查询人员关联的标签 ID 列表。
func (r *PersonTagRelationRepository) ListTagIDsByPerson(ctx context.Context, personID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&model.PersonTagRelation{}).Where("person_record_id = ?", personID).Pluck("tag_id", &ids).Error
	return ids, err
}
