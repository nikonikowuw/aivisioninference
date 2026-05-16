// Package main provides the database migration command for niko-admin.
package main

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/config"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/database"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name)

	db, err := database.New(dsn, cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	// Auto migrate all models.
	if err := db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.AuditLog{},
		&model.File{},
		&model.FileChunk{},
		&model.Task{},
		&model.UserRole{},
		&model.RolePermission{},
	); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// Seed default data.
	seedData(db, cfg.Seed)

	log.Println("Migration completed successfully")
}

// seedData inserts the default admin user, role, and permissions if they do not exist.
func seedData(db *gorm.DB, seedCfg config.SeedConfig) {
	// Check if admin user exists.
	var count int64
	db.Model(&model.User{}).Where("username = ?", seedCfg.Username).Count(&count)
	if count > 0 {
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// Create admin user.
		hashedPwd, err := hash.Hash(seedCfg.Password)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		admin := model.User{
			Username:    seedCfg.Username,
			Password:    hashedPwd,
			Email:       seedCfg.Email,
			DisplayName: seedCfg.DisplayName,
			Status:      1,
		}
		if err := tx.Create(&admin).Error; err != nil {
			return fmt.Errorf("create admin: %w", err)
		}

		// Create admin role.
		role := model.Role{
			Name:        "admin",
			Description: "系统管理员",
			SortOrder:   1,
			Status:      1,
		}
		if err := tx.Create(&role).Error; err != nil {
			return fmt.Errorf("create admin role: %w", err)
		}

		// Assign admin role to admin user.
		if err := tx.Model(&admin).Association("Roles").Append(&role); err != nil {
			return fmt.Errorf("assign admin role: %w", err)
		}

		// Create menu permissions and their button children.
		// Button permissions carry API path+method for RBAC middleware checks.
		type buttonInfo struct {
			Code   string
			Name   string
			Path   string // API path for RBAC (empty = UI-only)
			Method string // HTTP method for RBAC
		}
		menuList := []struct {
			Name    string
			Code    string
			Path    string // frontend route
			Icon    string
			Buttons []buttonInfo
		}{
			{Name: "仪表盘", Code: "dashboard", Path: "/default", Icon: "MdHome", Buttons: []buttonInfo{
				{Code: "dashboard:view", Name: "查看仪表盘", Path: "/api/v1/dashboard/stats", Method: "GET"},
			}},
			{Name: "用户管理", Code: "users", Path: "/users", Icon: "MdPerson", Buttons: []buttonInfo{
				{Code: "user:create", Name: "创建用户", Path: "/api/v1/users", Method: "POST"},
				{Code: "user:edit", Name: "编辑用户", Path: "/api/v1/users/*", Method: "PUT"},
				{Code: "user:delete", Name: "删除用户", Path: "/api/v1/users/*", Method: "DELETE"},
				{Code: "user:view", Name: "查看用户", Path: "/api/v1/users/*", Method: "GET"},
			}},
			{Name: "角色管理", Code: "roles", Path: "/roles", Icon: "MdSecurity", Buttons: []buttonInfo{
				{Code: "role:create", Name: "创建角色", Path: "/api/v1/roles", Method: "POST"},
				{Code: "role:edit", Name: "编辑角色", Path: "/api/v1/roles/*", Method: "PUT"},
				{Code: "role:delete", Name: "删除角色", Path: "/api/v1/roles/*", Method: "DELETE"},
				{Code: "role:assign-permissions", Name: "分配权限", Path: "/api/v1/roles/*/permissions", Method: "PUT"},
				{Code: "role:view-permissions", Name: "查看角色权限", Path: "/api/v1/roles/*/permissions", Method: "GET"},
				{Code: "role:view", Name: "查看角色", Path: "/api/v1/roles/*", Method: "GET"},
			}},
			{Name: "权限管理", Code: "permissions", Path: "/permissions", Icon: "MdVpnKey", Buttons: []buttonInfo{
				{Code: "permission:create", Name: "创建权限", Path: "/api/v1/permissions", Method: "POST"},
				{Code: "permission:edit", Name: "编辑权限", Path: "/api/v1/permissions/*", Method: "PUT"},
				{Code: "permission:delete", Name: "删除权限", Path: "/api/v1/permissions/*", Method: "DELETE"},
				{Code: "permission:view", Name: "查看权限", Path: "/api/v1/permissions/*", Method: "GET"},
			}},
			{Name: "文件管理", Code: "files", Path: "/files", Icon: "MdFolder", Buttons: []buttonInfo{
				{Code: "file:upload", Name: "上传文件", Path: "/api/v1/files/upload/**", Method: "POST"},
				{Code: "file:check", Name: "校验文件", Path: "/api/v1/files/upload/check", Method: "POST"},
				{Code: "file:upload-progress", Name: "上传进度", Path: "/api/v1/files/upload/*/progress", Method: "GET"},
				{Code: "file:delete", Name: "删除文件", Path: "/api/v1/files/*", Method: "DELETE"},
				{Code: "file:view", Name: "查看文件", Path: "/api/v1/files/*", Method: "GET"},
				{Code: "file:download", Name: "下载文件", Path: "/api/v1/files/*/download", Method: "GET"},
			}},
			{Name: "审计日志", Code: "audit-logs", Path: "/audit-logs", Icon: "MdHistory", Buttons: []buttonInfo{
				{Code: "audit:view", Name: "查看审计日志", Path: "/api/v1/audit-logs", Method: "GET"},
			}},
			{Name: "任务管理", Code: "tasks", Path: "/tasks", Icon: "MdAssignment", Buttons: []buttonInfo{
				{Code: "task:create", Name: "创建任务", Path: "/api/v1/tasks", Method: "POST"},
				{Code: "task:cancel", Name: "取消任务", Path: "/api/v1/tasks/*/cancel", Method: "POST"},
				{Code: "task:view", Name: "查看任务", Path: "/api/v1/tasks/*", Method: "GET"},
			}},
		}

		var allPerms []model.Permission
		for i, m := range menuList {
			menu := model.Permission{
				Name:      m.Name,
				Code:      m.Code,
				Path:      m.Path,
				Icon:      m.Icon,
				Type:      "menu",
				SortOrder: i + 1,
			}
			if err := tx.Create(&menu).Error; err != nil {
				return fmt.Errorf("create menu %s: %w", m.Code, err)
			}
			allPerms = append(allPerms, menu)

			// Create button permissions as children
			for _, btn := range m.Buttons {
				btnPerm := model.Permission{
					Name:      btn.Name,
					Code:      btn.Code,
					Path:      btn.Path,
					Method:    btn.Method,
					Type:      "button",
					ParentID:  &menu.ID,
					SortOrder: 0,
				}
				if err := tx.Create(&btnPerm).Error; err != nil {
					return fmt.Errorf("create button %s: %w", btn.Code, err)
				}
				allPerms = append(allPerms, btnPerm)
			}
		}

		// Assign all permissions to admin role.
		if err := tx.Model(&role).Association("Permissions").Append(allPerms); err != nil {
			return fmt.Errorf("assign permissions: %w", err)
		}

		return nil
	})
	if err != nil {
		log.Printf("seed data failed: %v", err)
		return
	}

	log.Printf("Default data seeded successfully")
}
