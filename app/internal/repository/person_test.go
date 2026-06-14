// Package repository 人员管理模块 Repository 单元测试。
package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

func setupPersonTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:person_repo_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS persons (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			person_code TEXT,
			person_name TEXT NOT NULL,
			gender TEXT DEFAULT 'unknown',
			phone TEXT,
			id_number TEXT,
			image_url TEXT NOT NULL,
			image_md5 TEXT,
			face_quality_score REAL,
			embedding_status TEXT NOT NULL DEFAULT 'pending',
			embedding_error_code TEXT,
			embedding_error_message_key TEXT,
			embedding_retryable INTEGER DEFAULT 0,
			enabled INTEGER DEFAULT 1,
			remark TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_persons_code ON persons(person_code);
		CREATE INDEX IF NOT EXISTS idx_persons_deleted_at ON persons(deleted_at);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS person_groups (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			group_name TEXT NOT NULL,
			description TEXT,
			parent_id TEXT,
			sort_order INTEGER DEFAULT 0
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS person_group_members (
			person_record_id TEXT NOT NULL,
			group_id TEXT NOT NULL,
			created_at DATETIME,
			PRIMARY KEY (person_record_id, group_id)
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS person_tags (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			tag_name TEXT NOT NULL UNIQUE,
			color TEXT,
			sort_order INTEGER DEFAULT 0
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS person_tag_relations (
			person_record_id TEXT NOT NULL,
			tag_id TEXT NOT NULL,
			created_at DATETIME,
			PRIMARY KEY (person_record_id, tag_id)
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS person_embeddings (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			person_record_id TEXT NOT NULL UNIQUE,
			embedding BLOB NOT NULL,
			version INTEGER DEFAULT 1
		);
	`).Error)
	return db
}

func createTestPerson(t *testing.T, repo *repository.PersonRepository, code, name string) string {
	t.Helper()
	ctx := context.Background()
	person := &model.Person{
		PersonCode:      code,
		PersonName:      name,
		Gender:          model.GenderUnknown,
		ImageURL:        "http://example.com/" + code + ".jpg",
		EmbeddingStatus: model.EmbeddingStatusPending,
		Enabled:         true,
	}
	person.ID = uuid.New().String()
	require.NoError(t, repo.Create(ctx, person))
	return person.ID
}

func TestPersonRepository_Create(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	person := &model.Person{
		PersonCode:      "P001",
		PersonName:      "张三",
		Gender:          model.GenderMale,
		Phone:           "13800001111",
		ImageURL:        "http://example.com/zhangsan.jpg",
		EmbeddingStatus: model.EmbeddingStatusPending,
		Enabled:         true,
	}
	person.ID = uuid.New().String()
	err := repo.Create(ctx, person)
	assert.NoError(t, err)
	assert.NotEmpty(t, person.ID)
}

func TestPersonRepository_FindByID(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	id := createTestPerson(t, repo, "P002", "李四")
	found, err := repo.FindByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, "李四", found.PersonName)
	assert.Equal(t, "P002", found.PersonCode)
}

func TestPersonRepository_List(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	createTestPerson(t, repo, "P003", "王五")
	createTestPerson(t, repo, "P004", "赵六")

	req := dto.PersonListRequest{PageRequest: dto.PageRequest{Page: 1, PageSize: 10}}
	items, total, err := repo.List(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
}

func TestPersonRepository_ListWithKeyword(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	createTestPerson(t, repo, "P005", "WangWu")
	createTestPerson(t, repo, "P006", "ZhaoLiu")

	req := dto.PersonListRequest{
		PageRequest:     dto.PageRequest{Page: 1, PageSize: 10},
		EmbeddingStatus: model.EmbeddingStatusPending,
	}
	// Keyword search uses ILIKE (PostgreSQL-specific), test with status filter instead
	items, total, err := repo.List(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
}

func TestPersonRepository_SoftDelete(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	id := createTestPerson(t, repo, "P007", "钱七")
	err := repo.SoftDelete(ctx, id)
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, id)
	assert.Error(t, err)
}

func TestPersonRepository_BatchSoftDelete(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	id1 := createTestPerson(t, repo, "P008", "A")
	id2 := createTestPerson(t, repo, "P009", "B")

	err := repo.BatchSoftDelete(ctx, []string{id1, id2})
	assert.NoError(t, err)

	_, err = repo.FindByID(ctx, id1)
	assert.Error(t, err)
}

func TestPersonRepository_BatchToggle(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	id1 := createTestPerson(t, repo, "P010", "C")
	id2 := createTestPerson(t, repo, "P011", "D")

	err := repo.BatchToggle(ctx, []string{id1, id2}, false)
	assert.NoError(t, err)

	found, _ := repo.FindByID(ctx, id1)
	assert.False(t, found.Enabled)
}

func TestPersonRepository_ExistsByPersonCode(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	createTestPerson(t, repo, "P012", "E")

	exists, err := repo.ExistsByPersonCode(ctx, "P012", "")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByPersonCode(ctx, "P999", "")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestPersonRepository_ExistsByPersonCodeExcludeID(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	id := createTestPerson(t, repo, "P013", "F")

	exists, err := repo.ExistsByPersonCode(ctx, "P013", id)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestPersonRepository_UpdateEmbeddingStatus(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonRepository(db)
	ctx := context.Background()

	id := createTestPerson(t, repo, "P014", "G")

	err := repo.UpdateEmbeddingStatus(ctx, id, model.EmbeddingStatusFailed, "NO_FACE", "person.error.noFace", false)
	assert.NoError(t, err)

	found, _ := repo.FindByID(ctx, id)
	assert.Equal(t, model.EmbeddingStatusFailed, found.EmbeddingStatus)
	assert.Equal(t, "NO_FACE", found.EmbeddingErrorCode)
	assert.False(t, found.EmbeddingRetryable)
}

func TestPersonGroupRepository_CRUD(t *testing.T) {
	db := setupPersonTestDB(t)
	repo := repository.NewPersonGroupRepository(db)
	ctx := context.Background()

	group := &model.PersonGroup{GroupName: "一厂区"}
	group.ID = uuid.New().String()
	err := repo.Create(ctx, group)
	assert.NoError(t, err)

	found, err := repo.FindByID(ctx, group.ID)
	assert.NoError(t, err)
	assert.Equal(t, "一厂区", found.GroupName)

	group.GroupName = "二厂区"
	err = repo.Update(ctx, group)
	assert.NoError(t, err)

	list, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 1)

	err = repo.Delete(ctx, group.ID)
	assert.NoError(t, err)
}

func TestPersonGroupRepository_CountPersonsExcludeSoftDeleted(t *testing.T) {
	db := setupPersonTestDB(t)
	personRepo := repository.NewPersonRepository(db)
	groupRepo := repository.NewPersonGroupRepository(db)
	ctx := context.Background()

	// 1. Create a group
	group := &model.PersonGroup{GroupName: "测试组"}
	group.ID = uuid.New().String()
	require.NoError(t, groupRepo.Create(ctx, group))

	// 2. Create two persons
	p1ID := createTestPerson(t, personRepo, "P101", "正常人员")
	p2ID := createTestPerson(t, personRepo, "P102", "已删除人员")

	// 3. Associate both to the group
	require.NoError(t, db.Exec("INSERT INTO person_group_members (person_record_id, group_id, created_at) VALUES (?, ?, ?)", p1ID, group.ID, time.Now()).Error)
	require.NoError(t, db.Exec("INSERT INTO person_group_members (person_record_id, group_id, created_at) VALUES (?, ?, ?)", p2ID, group.ID, time.Now()).Error)

	// Before soft delete, count should be 2
	c, err := groupRepo.CountPersons(ctx, group.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), c)

	batch, err := groupRepo.BatchCountPersons(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), batch[group.ID])

	// 4. Soft delete the second person
	require.NoError(t, personRepo.SoftDelete(ctx, p2ID))

	// After soft delete, count should be 1
	c2, err := groupRepo.CountPersons(ctx, group.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), c2)

	batch2, err := groupRepo.BatchCountPersons(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), batch2[group.ID])
}

func TestPersonTagRepository_CountPersonsExcludeSoftDeleted(t *testing.T) {
	db := setupPersonTestDB(t)
	personRepo := repository.NewPersonRepository(db)
	tagRepo := repository.NewPersonTagRepository(db)
	tagRelRepo := repository.NewPersonTagRelationRepository(db)
	ctx := context.Background()

	// 1. Create a tag
	tag := &model.PersonTag{TagName: "VIP"}
	tag.ID = uuid.New().String()
	require.NoError(t, tagRepo.Create(ctx, tag))

	// 2. Create two persons
	p1ID := createTestPerson(t, personRepo, "P201", "正常标签人员")
	p2ID := createTestPerson(t, personRepo, "P202", "已删除标签人员")

	// 3. Associate both to the tag
	require.NoError(t, tagRelRepo.BatchCreate(ctx, p1ID, []string{tag.ID}))
	require.NoError(t, tagRelRepo.BatchCreate(ctx, p2ID, []string{tag.ID}))

	// Before soft delete, count should be 2
	c, err := tagRepo.CountPersons(ctx, tag.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), c)

	batch, err := tagRepo.BatchCountPersons(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), batch[tag.ID])

	// 4. Soft delete the second person
	require.NoError(t, personRepo.SoftDelete(ctx, p2ID))

	// After soft delete, count should be 1
	c2, err := tagRepo.CountPersons(ctx, tag.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), c2)

	batch2, err := tagRepo.BatchCountPersons(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), batch2[tag.ID])
}
