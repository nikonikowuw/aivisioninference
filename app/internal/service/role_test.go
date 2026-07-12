package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

func TestRoleServiceCreateRejectsMissingLevelWithRoleLevelMessage(t *testing.T) {
	svc := &RoleService{}

	role, err := svc.Create(context.Background(), dto.CreateRoleRequest{Name: "manager"}, "", true)

	require.Nil(t, role)
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.ErrRoleLevelInvalid, appErr.Code)
	assert.Equal(t, apperrors.New(apperrors.ErrRoleLevelInvalid, "").Message, appErr.Message)
}

func TestRoleHierarchyLevelRoleMessageExplainsLevelChangeRule(t *testing.T) {
	err := apperrors.New(apperrors.ErrHierarchyLevelRole, "")

	assert.Equal(t, apperrors.ErrHierarchyLevelRole, err.Code)
	// Tests always run in default language (en) since New() uses defaultLanguage ("en")
	assert.Equal(t, "No permission to operate on roles at or above your level", err.Message)
}

func TestCheckRoleRootGuardRejectsNonRootSystemRoleChanges(t *testing.T) {
	tests := []struct {
		name      string
		roleLevel int
		isRoot    bool
		wantErr   bool
	}{
		{name: "non-root rejects admin name", roleLevel: 1, isRoot: false, wantErr: true},
		{name: "non-root rejects renamed level-1 role", roleLevel: 1, isRoot: false, wantErr: true},
		{name: "root allows level-1 role", roleLevel: 1, isRoot: true, wantErr: false},
		{name: "non-root allows non-system role", roleLevel: 2, isRoot: false, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkRoleRootGuard(tt.roleLevel, tt.isRoot)
			if tt.wantErr {
				require.Error(t, err)
				var appErr *apperrors.AppError
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, apperrors.ErrHierarchyLevelRole, appErr.Code)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRoleServiceRootGuardOnUpdateDeleteAssign(t *testing.T) {
	tests := []struct {
		name              string
		action            string
		isRoot            bool
		wantErr           bool
		wantCode          string
		roleName          string
		roleLevel         int
		seedCurrentUserID string
		seedCurrentLevel  int
	}{
		{
			name:      "update should reject non-root operating admin role",
			action:    "update",
			isRoot:    false,
			wantErr:   true,
			wantCode:  apperrors.ErrHierarchyLevelRole,
			roleName:  "admin",
			roleLevel: 1,
		},
		{
			name:      "delete should reject non-root operating admin role",
			action:    "delete",
			isRoot:    false,
			wantErr:   true,
			wantCode:  apperrors.ErrHierarchyLevelRole,
			roleName:  "admin",
			roleLevel: 1,
		},
		{
			name:      "assign permissions should reject non-root operating admin role",
			action:    "assign",
			isRoot:    false,
			wantErr:   true,
			wantCode:  apperrors.ErrHierarchyLevelRole,
			roleName:  "admin",
			roleLevel: 1,
		},
		{
			name:              "update should reject non-root operating level-1 role even when renamed",
			action:            "update",
			isRoot:            false,
			wantErr:           true,
			wantCode:          apperrors.ErrHierarchyLevelRole,
			roleName:          "super-admin",
			roleLevel:         1,
			seedCurrentUserID: "user-1",
			seedCurrentLevel:  2,
		},
		{
			name:              "delete should reject non-root operating level-1 role even when renamed",
			action:            "delete",
			isRoot:            false,
			wantErr:           true,
			wantCode:          apperrors.ErrHierarchyLevelRole,
			roleName:          "super-admin",
			roleLevel:         1,
			seedCurrentUserID: "user-1",
			seedCurrentLevel:  2,
		},
		{
			name:              "assign permissions should reject non-root operating level-1 role even when renamed",
			action:            "assign",
			isRoot:            false,
			wantErr:           true,
			wantCode:          apperrors.ErrHierarchyLevelRole,
			roleName:          "super-admin",
			roleLevel:         1,
			seedCurrentUserID: "user-1",
			seedCurrentLevel:  2,
		},
		{
			name:      "update should allow root operating admin role",
			action:    "update",
			isRoot:    true,
			wantErr:   false,
			roleName:  "admin",
			roleLevel: 1,
		},
		{
			name:      "delete should allow root operating admin role",
			action:    "delete",
			isRoot:    true,
			wantErr:   false,
			roleName:  "admin",
			roleLevel: 1,
		},
		{
			name:      "assign permissions should allow root operating admin role",
			action:    "assign",
			isRoot:    true,
			wantErr:   false,
			roleName:  "admin",
			roleLevel: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupRoleServiceTestDB(t)
			ctx := context.Background()
			roleID := fmt.Sprintf("role-%s", tt.name)
			roleRepo := repository.NewRoleRepository(db)
			userRepo := repository.NewUserRepository(db)
			svc := NewRoleService(roleRepo, userRepo, nil)

			role := model.Role{
				BaseModel:   model.BaseModel{ID: roleID},
				Name:        tt.roleName,
				Description: "system role",
				SortOrder:   1,
				Status:      1,
				Level:       tt.roleLevel,
			}
			require.NoError(t, db.Create(&role).Error)
			if tt.seedCurrentUserID != "" {
				require.NoError(t, db.Exec("INSERT INTO roles (id, name, level, status, sort_order) VALUES (?, ?, ?, ?, ?)", "role-current", "current-user-role", tt.seedCurrentLevel, 1, 0).Error)
				require.NoError(t, db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", tt.seedCurrentUserID, "role-current").Error)
			}

			currentUserID := tt.seedCurrentUserID
			var err error
			switch tt.action {
			case "update":
				newName := "updated-name"
				err = svc.Update(ctx, role.ID, dto.UpdateRoleRequest{Name: newName}, currentUserID, tt.isRoot)
			case "delete":
				err = svc.Delete(ctx, role.ID, currentUserID, tt.isRoot)
			case "assign":
				err = svc.AssignPermissions(ctx, role.ID, dto.AssignPermissionsRequest{PermissionIDs: []string{}}, currentUserID, tt.isRoot)
			default:
				t.Fatalf("unsupported action: %s", tt.action)
			}

			if tt.wantErr {
				require.Error(t, err)
				var appErr *apperrors.AppError
				require.ErrorAs(t, err, &appErr)
				assert.Equal(t, tt.wantCode, appErr.Code)
				return
			}

			require.NoError(t, err)
		})
	}
}

func setupRoleServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:role_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
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
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS user_roles (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL
		);
	`).Error)
	return db
}
