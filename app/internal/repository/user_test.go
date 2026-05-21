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

func setupUserRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:user_repo_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			email TEXT UNIQUE,
			email_verified INTEGER DEFAULT 0,
			display_name TEXT,
			avatar_url TEXT,
			status INTEGER DEFAULT 1,
			login_attempts INTEGER DEFAULT 0,
			locked_until DATETIME,
			is_root INTEGER DEFAULT 0
		);
	`).Error)
	return db
}

func TestCountByEmail(t *testing.T) {
	db := setupUserRepositoryTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	users := []model.User{
		{BaseModel: model.BaseModel{ID: "user-1"}, Username: "alice", Email: "alice@example.com", Password: "x"},
		{BaseModel: model.BaseModel{ID: "user-2"}, Username: "bob", Email: "bob@example.com", Password: "x"},
		{BaseModel: model.BaseModel{ID: "user-3"}, Username: "carol", Email: "", Password: "x"},
	}
	for _, u := range users {
		require.NoError(t, db.Create(&u).Error)
	}

	tests := []struct {
		name      string
		email     string
		excludeID string
		want      int64
	}{
		{name: "existing email no exclude", email: "alice@example.com", excludeID: "", want: 1},
		{name: "existing email exclude self", email: "alice@example.com", excludeID: "user-1", want: 0},
		{name: "nonexistent email", email: "nobody@example.com", excludeID: "", want: 0},
		{name: "case insensitive", email: "Alice@Example.COM", excludeID: "", want: 1},
		{name: "empty email excluded", email: "", excludeID: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.CountByEmail(ctx, tt.email, tt.excludeID)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
