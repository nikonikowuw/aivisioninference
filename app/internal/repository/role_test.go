package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

func TestRoleRepositoryUpdateUsesExplicitPrimaryKeyFilter(t *testing.T) {
	db := setupRoleRepositoryTestDB(t)
	repo := NewRoleRepository(db)

	role := model.Role{
		BaseModel: model.BaseModel{ID: "role-1"},
		Name:      "admin",
		Description: "before",
		SortOrder:   1,
		Status:      1,
		Level:       1,
	}
	require.NoError(t, db.Create(&role).Error)

	updated := model.Role{
		BaseModel: model.BaseModel{ID: role.ID, DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true}},
		Name:      "after",
		Description: "updated",
		SortOrder:   2,
		Status:      1,
		Level:       1,
	}
	require.NoError(t, repo.Update(context.Background(), &updated))

	var got model.Role
	require.NoError(t, db.First(&got, "id = ?", role.ID).Error)
	require.Equal(t, "after", got.Name)
	require.Equal(t, "updated", got.Description)
	require.Equal(t, 2, got.SortOrder)
}

func TestRoleRepositoryUpdateWithEmptyIDDoesNotUpdateAnyRecord(t *testing.T) {
	db := setupRoleRepositoryTestDB(t)
	repo := NewRoleRepository(db)

	role := model.Role{
		BaseModel: model.BaseModel{ID: "role-1"},
		Name:      "admin",
		Description: "before",
		SortOrder:   1,
		Status:      1,
		Level:       1,
	}
	require.NoError(t, db.Create(&role).Error)

	item := &model.Role{
		Name:        "new-name",
		Description: "new-desc",
		SortOrder:   9,
		Status:      1,
		Level:       1,
	}
	require.NoError(t, repo.Update(context.Background(), item))

	var got model.Role
	require.NoError(t, db.First(&got, "id = ?", role.ID).Error)
	require.Equal(t, "admin", got.Name)
	require.Equal(t, "before", got.Description)
	require.Equal(t, 1, got.SortOrder)
}

func setupRoleRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:role_repo_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			name TEXT NOT NULL,
			description TEXT,
			sort_order INTEGER DEFAULT 0,
			status INTEGER DEFAULT 1,
			level INTEGER DEFAULT 100
		);
	`).Error)
	return db
}
