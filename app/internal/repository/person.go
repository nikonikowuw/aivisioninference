// Package repository 提供人员管理模块的数据访问层。
package repository

import (
	"context"
	"time"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
	"github.com/pgvector/pgvector-go"
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

// buildListQuery 构建查询基础（筛选 + 分组过滤），用于 List 和 ListForExport。
func (r *PersonRepository) buildListQuery(ctx context.Context, req dto.PersonListRequest) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&model.Person{}).Scopes(req.FilterScopes()...)
	if req.GroupID != "" {
		q = q.Joins("JOIN person_group_members pgm ON pgm.person_record_id = persons.id AND pgm.group_id = ?", req.GroupID)
	}
	return q
}

// List 查询人员分页列表。
func (r *PersonRepository) List(ctx context.Context, req dto.PersonListRequest) ([]model.Person, int64, error) {
	base := r.buildListQuery(ctx, req)
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
	q := r.buildListQuery(ctx, req)
	return items, q.Scopes(scopes.OrderBy(req.Sort, req.Order, model.Person{}.SortableFields()...), scopes.OrderByDefault()).Limit(limit).Preload("Groups").Find(&items).Error
}

// Update 更新人员记录，仅更新业务字段（不覆盖零值）。
func (r *PersonRepository) Update(ctx context.Context, item *model.Person) error {
	return r.db.WithContext(ctx).Model(item).Select(
		"person_code", "person_name", "gender", "phone", "id_number",
		"image_url", "image_md5", "face_quality_score", "embedding_status",
		"embedding_error_code", "embedding_error_message_key", "embedding_retryable",
		"enabled", "remark",
	).Updates(item).Error
}

// SoftDelete 软删除人员。同时清除图片 MD5 和 URL，释放唯一索引，
// 允许后续使用同一张图片重新创建人员。
func (r *PersonRepository) SoftDelete(ctx context.Context, id string) error {
	return r.BatchSoftDelete(ctx, []string{id})
}

// BatchSoftDelete 批量软删除人员。
func (r *PersonRepository) BatchSoftDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&model.Person{}).Where("id IN ?", ids).Updates(map[string]interface{}{
		"deleted_at": time.Now(),
		"image_md5":  nil,
		"image_url":  "",
	}).Error
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
	if code == "" {
		return false, nil
	}
	q := r.db.WithContext(ctx).Model(&model.Person{}).Where("person_code = ?", code)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

// ExistsByImageMD5 检查图片 MD5 是否存在。
func (r *PersonRepository) ExistsByImageMD5(ctx context.Context, md5, excludeID string) (bool, error) {
	if md5 == "" {
		return false, nil
	}
	q := r.db.WithContext(ctx).Model(&model.Person{}).Where("image_md5 = ?", md5)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
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
		members := make([]model.PersonGroupMember, 0, len(groupIDs))
		for _, id := range groupIDs {
			if id != "" {
				members = append(members, model.PersonGroupMember{PersonRecordID: personID, GroupID: id})
			}
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
func NewPersonGroupRepository(db *gorm.DB) *PersonGroupRepository {
	return &PersonGroupRepository{db: db}
}

func (r *PersonGroupRepository) Create(ctx context.Context, item *model.PersonGroup) error {
	return r.db.WithContext(ctx).Create(item).Error
}
func (r *PersonGroupRepository) FindByID(ctx context.Context, id string) (*model.PersonGroup, error) {
	var item model.PersonGroup
	err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error
	return &item, err
}
func (r *PersonGroupRepository) Update(ctx context.Context, item *model.PersonGroup) error {
	return r.db.WithContext(ctx).Save(item).Error
}
func (r *PersonGroupRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.PersonGroup{}, "id = ?", id).Error
}
func (r *PersonGroupRepository) List(ctx context.Context) ([]model.PersonGroup, error) {
	var items []model.PersonGroup
	err := r.db.WithContext(ctx).Order("sort_order ASC").Order("created_at DESC").Find(&items).Error
	return items, err
}
func (r *PersonGroupRepository) CountPersons(ctx context.Context, id string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("person_group_members").
		Joins("JOIN persons ON persons.id = person_group_members.person_record_id").
		Where("person_group_members.group_id = ? AND persons.deleted_at IS NULL", id).
		Count(&count).Error
	return count, err
}
func (r *PersonGroupRepository) BatchCountPersons(ctx context.Context) (map[string]int64, error) {
	counts := make(map[string]int64)
	var rows []struct {
		GroupID string `gorm:"column:group_id"`
		Count   int64  `gorm:"column:count"`
	}
	err := r.db.WithContext(ctx).
		Table("person_group_members").
		Select("person_group_members.group_id, COUNT(*) as count").
		Joins("JOIN persons ON persons.id = person_group_members.person_record_id").
		Where("persons.deleted_at IS NULL").
		Group("person_group_members.group_id").
		Find(&rows).Error
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

// FaceEmbeddingRecord 表示可下发给算法引擎的人脸库记录。
type FaceEmbeddingRecord struct {
	PersonID   string          `gorm:"column:person_id"`
	PersonName string          `gorm:"column:person_name"`
	Embedding  pgvector.Vector `gorm:"column:embedding"`
}

// NewPersonEmbeddingRepository 创建人员向量 Repository。
func NewPersonEmbeddingRepository(db *gorm.DB) *PersonEmbeddingRepository {
	return &PersonEmbeddingRepository{db: db}
}

func (r *PersonEmbeddingRepository) DeleteByPersonRecordID(ctx context.Context, personID string) error {
	return r.db.WithContext(ctx).Where("person_record_id = ?", personID).Unscoped().Delete(&model.PersonEmbedding{}).Error
}
func (r *PersonEmbeddingRepository) Create(ctx context.Context, item *model.PersonEmbedding) error {
	return r.db.WithContext(ctx).Create(item).Error
}
func (r *PersonEmbeddingRepository) ListActive(ctx context.Context, limit int) ([]model.PersonEmbedding, error) {
	var items []model.PersonEmbedding
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	err := r.db.WithContext(ctx).Limit(limit).Find(&items).Error
	return items, err
}

func (r *PersonEmbeddingRepository) ListActiveFaceLibrary(ctx context.Context, limit int) ([]FaceEmbeddingRecord, error) {
	if limit <= 0 || limit > 100000 {
		limit = 100000
	}
	var items []FaceEmbeddingRecord
	err := r.db.WithContext(ctx).
		Table("person_embeddings pe").
		Select("p.id AS person_id, p.person_name, pe.embedding").
		Joins("JOIN persons p ON p.id = pe.person_record_id").
		Where("p.enabled = ? AND p.embedding_status = ? AND p.deleted_at IS NULL", true, model.EmbeddingStatusActive).
		Order("p.updated_at DESC").
		Limit(limit).
		Scan(&items).Error
	return items, err
}

// PersonSearchByFaceRecord 以图搜人查询结果行。
type PersonSearchByFaceRecord struct {
	model.Person
	Distance float64 `gorm:"column:distance"`
}

// SearchByFace 执行 pgvector 余弦距离搜索，按相似度降序返回 active/enabled 人员。
// queryVector 为查询图提取的 512 维特征，topK 上限 50，threshold 为最低余弦相似度(0-1)。
func (r *PersonEmbeddingRepository) SearchByFace(ctx context.Context, queryVector pgvector.Vector, topK int, threshold float64) ([]PersonSearchByFaceRecord, error) {
	if topK <= 0 || topK > 50 {
		topK = 5
	}
	if threshold < 0 || threshold > 1 {
		threshold = 0.5
	}
	// threshold 为缩放后相似度 [0,1]
	// distance = 1 - raw_cos, scaled_cos = (raw_cos + 1)/2 = (1 - distance + 1)/2 = (2 - distance)/2
	// maxDistance = 2 - 2*threshold
	maxDistance := 2.0 - 2.0*threshold

	// 第一步：查询匹配的人员 ID 和余弦距离，不通过 HAVING 引用 SELECT alias
	type idDistance struct {
		PersonID string  `gorm:"column:person_id"`
		Distance float64 `gorm:"column:distance"`
	}
	var ids []idDistance
	err := r.db.WithContext(ctx).
		Table("person_embeddings pe").
		Select("pe.person_record_id AS person_id, pe.embedding <=> ? AS distance", queryVector).
		Joins("JOIN persons p ON p.id = pe.person_record_id").
		Where("p.enabled = ? AND p.embedding_status = ? AND p.deleted_at IS NULL", true, model.EmbeddingStatusActive).
		Where("(pe.embedding <=> ?) <= ?", queryVector, maxDistance).
		Order("distance ASC").
		Limit(topK).
		Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}

	// 第二步：提取 person IDs 并查询完整记录（含 Groups 预加载）
	personIDs := make([]string, len(ids))
	for i, v := range ids {
		personIDs[i] = v.PersonID
	}
	var persons []model.Person
	if err := r.db.WithContext(ctx).
		Preload("Groups").
		Where("id IN ?", personIDs).
		Find(&persons).Error; err != nil {
		return nil, err
	}

	// 第三步：按 distance 顺序组装结果
	personMap := make(map[string]*model.Person, len(persons))
	for i := range persons {
		personMap[persons[i].ID] = &persons[i]
	}
	results := make([]PersonSearchByFaceRecord, 0, len(ids))
	for _, v := range ids {
		p, ok := personMap[v.PersonID]
		if !ok {
			continue
		}
		results = append(results, PersonSearchByFaceRecord{
			Person:   *p,
			Distance: v.Distance,
		})
	}
	return results, nil
}

func (r *PersonRepository) ListEnabledForEmbedding(ctx context.Context, limit int) ([]model.Person, error) {
	if limit <= 0 || limit > 100000 {
		limit = 100000
	}
	var items []model.Person
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("updated_at DESC").
		Limit(limit).
		Find(&items).Error
	return items, err
}

// ImportTaskRepository 处理导入任务持久化。
type ImportTaskRepository struct{ db *gorm.DB }

// NewImportTaskRepository 创建导入任务 Repository。
func NewImportTaskRepository(db *gorm.DB) *ImportTaskRepository { return &ImportTaskRepository{db: db} }

func (r *ImportTaskRepository) Create(ctx context.Context, item *model.ImportTask) error {
	return r.db.WithContext(ctx).Create(item).Error
}
func (r *ImportTaskRepository) FindByID(ctx context.Context, id string) (*model.ImportTask, error) {
	var item model.ImportTask
	err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error
	return &item, err
}
func (r *ImportTaskRepository) Update(ctx context.Context, item *model.ImportTask) error {
	return r.db.WithContext(ctx).Model(item).Select(
		"task_type", "file_name", "file_url", "total_rows",
		"success_rows", "failed_rows", "fail_detail_url", "status", "completed_at",
	).Updates(item).Error
}
func (r *ImportTaskRepository) List(ctx context.Context, req dto.PageRequest) ([]model.ImportTask, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&model.ImportTask{}).Where("task_type = ?", model.ImportTaskTypePerson)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.ImportTask
	err := q.Scopes(scopes.OrderBy(req.Sort, req.Order, model.ImportTask{}.SortableFields()...), scopes.OrderByDefault(), scopes.Paginate(req.GetPage(), req.GetPageSize())).Find(&items).Error
	return items, total, err
}

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
	err := r.db.WithContext(ctx).
		Table("person_tag_relations").
		Select("person_tag_relations.tag_id, COUNT(*) as count").
		Joins("JOIN persons ON persons.id = person_tag_relations.person_record_id").
		Where("persons.deleted_at IS NULL").
		Group("person_tag_relations.tag_id").
		Find(&rows).Error
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
	err := r.db.WithContext(ctx).
		Table("person_tag_relations").
		Joins("JOIN persons ON persons.id = person_tag_relations.person_record_id").
		Where("person_tag_relations.tag_id = ? AND persons.deleted_at IS NULL", id).
		Count(&count).Error
	return count, err
}

// PersonTagRelationRepository 处理人员与标签关联持久化。
type PersonTagRelationRepository struct{ db *gorm.DB }

// NewPersonTagRelationRepository 创建人员标签关联 Repository。
func NewPersonTagRelationRepository(db *gorm.DB) *PersonTagRelationRepository {
	return &PersonTagRelationRepository{db: db}
}

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
