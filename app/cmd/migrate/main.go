// Package main provides the database migration command for niko-admin.
package main

import (
	"errors"
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
		&model.MailConfig{},
		&model.EmailToken{},
		&model.InboundEmail{},
		&model.Feedback{},
		&model.UserRole{},
		&model.RolePermission{},
	); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// Partial unique index: only one root user allowed.
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_root ON users (is_root) WHERE is_root = true").Error; err != nil {
		log.Fatalf("create unique root index: %v", err)
	}
	if err := db.Exec(rootUsernameConstraintSQL()).Error; err != nil {
		log.Fatalf("create root username constraint: %v", err)
	}
	if err := db.Exec(ensureAuditLogSummaryColumnsSQL()).Error; err != nil {
		log.Fatalf("ensure audit summary columns: %v", err)
	}
	if err := db.Exec(dropAuditLogActionColumnSQL()).Error; err != nil {
		log.Fatalf("drop audit log action column: %v", err)
	}
	if err := db.Exec(migrateAuditLogResultSummarySQL()).Error; err != nil {
		log.Fatalf("migrate audit log result summary: %v", err)
	}
	if err := db.Exec(addAuditLogActionTypeColumnSQL()).Error; err != nil {
		log.Fatalf("add audit log action type column: %v", err)
	}

	// Migrate existing menus to multi-level structure.
	if err := migrateMultiLevelMenu(db); err != nil {
		log.Fatalf("migrate multi-level menu: %v", err)
	}

	// Seed default data.
	if err := seedData(db, cfg.Seed); err != nil {
		log.Fatalf("seed data: %v", err)
	}

	log.Println("Migration completed successfully")
}

// rootUsernameConstraintSQL 返回 root 用户名一致性的幂等约束语句。
func rootUsernameConstraintSQL() string {
	return `
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'chk_root_username'
	) THEN
		ALTER TABLE users ADD CONSTRAINT chk_root_username CHECK (is_root = false OR username = 'root');
	END IF;
END
$$;`
}

// ensureAuditLogSummaryColumnsSQL 返回审计摘要字段的幂等迁移语句。
func ensureAuditLogSummaryColumnsSQL() string {
	return `
	ALTER TABLE audit_logs
		ADD COLUMN IF NOT EXISTS result_summary varchar(255);`
}

// dropAuditLogActionColumnSQL 删除 audit_logs 中冗余的 action 列。
// 注意：此操作不可逆，执行前需确认无外部系统依赖该列。
func dropAuditLogActionColumnSQL() string {
	return `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name = 'audit_logs' AND column_name = 'action'
		) THEN
			ALTER TABLE audit_logs DROP COLUMN action;
		END IF;
	END
	$$;`
}

// migrateAuditLogResultSummarySQL 将旧格式的 result_summary 统一为 success/failed，
// 并删除冗余的 error_summary 列。
func migrateAuditLogResultSummarySQL() string {
	return `
	UPDATE audit_logs SET result_summary = 'failed'
	WHERE result_summary NOT IN ('success', 'failed') AND result_summary IS NOT NULL AND result_summary != '';

	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name = 'audit_logs' AND column_name = 'error_summary'
		) THEN
			ALTER TABLE audit_logs DROP COLUMN error_summary;
		END IF;
	END
	$$;`
}

// addAuditLogActionTypeColumnSQL 添加操作类型字段到审计日志表。
func addAuditLogActionTypeColumnSQL() string {
	return `
	ALTER TABLE audit_logs
		ADD COLUMN IF NOT EXISTS action_type varchar(128);`
}

func shouldCreateSeedAdmin(adminUsernameExists, adminEmailExists bool) bool {
	return !adminUsernameExists && !adminEmailExists
}

// migrateMultiLevelMenu 将已有的平铺菜单迁移为两级结构。
// 幂等操作：已存在的父菜单不会重复创建，子菜单的 ParentID 仅在为空时更新。
// 保持现有角色绑定不变。
func migrateMultiLevelMenu(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		parents := []struct {
			Code       string
			Name       string
			Icon       string
			ChildCodes []string
		}{
			{Code: "user-management", Name: "用户管理", Icon: "MdPeople", ChildCodes: []string{"users", "roles", "permissions"}},
			{Code: "system-management", Name: "系统管理", Icon: "MdSettings", ChildCodes: []string{"files", "audit-logs", "tasks"}},
		}

		for _, pm := range parents {
			// 查找或创建父菜单。
			var parent model.Permission
			err := tx.Where("code = ? AND type = ?", pm.Code, "menu").First(&parent).Error
			if err == gorm.ErrRecordNotFound {
				parent = model.Permission{
					Name: pm.Name, Code: pm.Code,
					Path: "/" + pm.Code, Icon: pm.Icon,
					Type: "menu", SortOrder: 0,
				}
				if createErr := tx.Create(&parent).Error; createErr != nil {
					return fmt.Errorf("create parent menu %s: %w", pm.Code, createErr)
				}
				log.Printf("Created parent menu: %s", pm.Code)
			} else if err != nil {
				return fmt.Errorf("query parent menu %s: %w", pm.Code, err)
			}

			// 更新子菜单的 ParentID（仅更新 type='menu' 且 ParentID 为空的记录）。
			result := tx.Model(&model.Permission{}).
				Where("code IN ? AND type = 'menu' AND parent_id IS NULL", pm.ChildCodes).
				Update("parent_id", parent.ID)
			if result.Error != nil {
				return fmt.Errorf("update children for %s: %w", pm.Code, result.Error)
			}
			if result.RowsAffected > 0 {
				log.Printf("Updated %d child menus under %s", result.RowsAffected, pm.Code)
			}
		}

		return nil
	})
}

// seedData inserts the default admin user, role, and permissions if they do not exist.
func seedData(db *gorm.DB, seedCfg config.SeedConfig) error {
	// Ensure root user exists.
	var rootCount int64
	if err := db.Model(&model.User{}).Where("username = ?", "root").Count(&rootCount).Error; err != nil {
		return fmt.Errorf("count root user: %w", err)
	}
	if rootCount == 0 {
		var roleCount int64
		if err := db.Model(&model.Role{}).Where("name = ?", "admin").Count(&roleCount).Error; err != nil {
			return fmt.Errorf("count admin role: %w", err)
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			rootPwd, err := hash.Hash(seedCfg.RootPassword)
			if err != nil {
				return fmt.Errorf("hash root password: %w", err)
			}
			root := model.User{Username: "root", Password: rootPwd, Email: seedCfg.RootEmail, DisplayName: "超级管理员", Status: 1, IsRoot: true}
			if err := tx.Create(&root).Error; err != nil {
				return fmt.Errorf("create root: %w", err)
			}
			if roleCount > 0 {
				var role model.Role
				if err := tx.Where("name = ?", "admin").First(&role).Error; err != nil {
					return fmt.Errorf("query admin role: %w", err)
				}
				if err := tx.Model(&root).Association("Roles").Append(&role); err != nil {
					return fmt.Errorf("assign admin role to root: %w", err)
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// Check if full seed has been done (admin user exists).
	var usernameCount int64
	if err := db.Model(&model.User{}).Where("username = ?", seedCfg.Username).Count(&usernameCount).Error; err != nil {
		return fmt.Errorf("count admin user: %w", err)
	}
	var emailCount int64
	if err := db.Model(&model.User{}).Where("email = ?", seedCfg.Email).Count(&emailCount).Error; err != nil {
		return fmt.Errorf("count admin email: %w", err)
	}
	if shouldCreateSeedAdmin(usernameCount > 0, emailCount > 0) {
		if err := db.Transaction(func(tx *gorm.DB) error {
			adminPwd, err := hash.Hash(seedCfg.Password)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}

			admin := model.User{Username: seedCfg.Username, Password: adminPwd, Email: seedCfg.Email, DisplayName: seedCfg.DisplayName, Status: 1}
			if err := tx.Create(&admin).Error; err != nil {
				return fmt.Errorf("create admin: %w", err)
			}

			// Create admin role.
			role := model.Role{
				Name:        "admin",
				Description: "系统管理员",
				SortOrder:   1,
				Status:      1,
				Level:       1,
			}
			if err := tx.Create(&role).Error; err != nil {
				return fmt.Errorf("create admin role: %w", err)
			}

			// Assign admin role to admin user.
			if err := tx.Model(&admin).Association("Roles").Append(&role); err != nil {
				return fmt.Errorf("assign admin role to admin: %w", err)
			}

			// Also assign admin role to existing root user.
			var root model.User
			if err := tx.Where("username = ?", "root").First(&root).Error; err == nil {
				if err := tx.Model(&root).Association("Roles").Append(&role); err != nil {
					return fmt.Errorf("assign admin role to existing root: %w", err)
				}
			}

			return nil
		}); err != nil {
			return fmt.Errorf("seed admin user: %w", err)
		}
	}

	// Sync permissions: always ensure all defined permissions exist and are assigned to admin role.
	if err := syncPermissions(db); err != nil {
		return fmt.Errorf("sync permissions: %w", err)
	}

	log.Printf("Default data seeded successfully")
	return nil
}

// buttonInfo 定义按钮权限的 API 路径和方法。
type buttonInfo struct {
	Code   string
	Name   string
	Path   string // API path for RBAC (empty = UI-only)
	Method string // HTTP method for RBAC
}

// childMenuDef 定义子菜单及其按钮权限。
type childMenuDef struct {
	Name    string
	Code    string
	Path    string
	Icon    string
	Buttons []buttonInfo
}

// parentMenuDef 定义父菜单（分组）及其子菜单。
type parentMenuDef struct {
	Name     string
	Code     string
	Path     string
	Icon     string
	Buttons  []buttonInfo   // 父菜单自身的按钮权限（如仪表盘）
	Children []childMenuDef // 子菜单，nil 表示无子菜单
}

// defaultMenuList 返回系统默认的菜单和按钮权限定义（两级结构）。
func defaultMenuList() []parentMenuDef {
	return []parentMenuDef{
		{
			Name: "仪表盘", Code: "dashboard", Path: "/default", Icon: "MdHome",
			Buttons: []buttonInfo{
				{Code: "dashboard:view", Name: "查看仪表盘", Path: "/api/v1/dashboard/stats", Method: "GET"},
			},
			Children: nil,
		},
		{
			Name: "用户管理", Code: "user-management", Path: "/user-management", Icon: "MdPeople",
			Children: []childMenuDef{
				{Name: "用户", Code: "users", Path: "/users", Icon: "MdPerson", Buttons: []buttonInfo{
					{Code: "user:create", Name: "创建用户", Path: "/api/v1/users", Method: "POST"},
					{Code: "user:edit", Name: "编辑用户", Path: "/api/v1/users/*", Method: "PUT"},
					{Code: "user:delete", Name: "删除用户", Path: "/api/v1/users/*", Method: "DELETE"},
					{Code: "user:view", Name: "查看用户", Path: "/api/v1/users/*", Method: "GET"},
					{Code: "user:reset-password", Name: "重置密码", Path: "/api/v1/users/*/password", Method: "PUT"},
					{Code: "user:upload-avatar", Name: "上传头像", Path: "/api/v1/users/*/avatar", Method: "POST"},
				}},
				{Name: "角色", Code: "roles", Path: "/roles", Icon: "MdSecurity", Buttons: []buttonInfo{
					{Code: "role:create", Name: "创建角色", Path: "/api/v1/roles", Method: "POST"},
					{Code: "role:edit", Name: "编辑角色", Path: "/api/v1/roles/*", Method: "PUT"},
					{Code: "role:delete", Name: "删除角色", Path: "/api/v1/roles/*", Method: "DELETE"},
					{Code: "role:assign-permissions", Name: "分配权限", Path: "/api/v1/roles/*/permissions", Method: "PUT"},
					{Code: "role:view-permissions", Name: "查看角色权限", Path: "/api/v1/roles/*/permissions", Method: "GET"},
					{Code: "role:view", Name: "查看角色", Path: "/api/v1/roles/*", Method: "GET"},
				}},
				{Name: "权限", Code: "permissions", Path: "/permissions", Icon: "MdVpnKey", Buttons: []buttonInfo{
					{Code: "permission:create", Name: "创建权限", Path: "/api/v1/permissions", Method: "POST"},
					{Code: "permission:edit", Name: "编辑权限", Path: "/api/v1/permissions/*", Method: "PUT"},
					{Code: "permission:delete", Name: "删除权限", Path: "/api/v1/permissions/*", Method: "DELETE"},
					{Code: "permission:view", Name: "查看权限", Path: "/api/v1/permissions/*", Method: "GET"},
				}},
			},
		},
		{
			Name: "系统管理", Code: "system-management", Path: "/system-management", Icon: "MdSettings",
			Children: []childMenuDef{
				{Name: "文件", Code: "files", Path: "/files", Icon: "MdFolder", Buttons: []buttonInfo{
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
				{Name: "任务", Code: "tasks", Path: "/tasks", Icon: "MdAssignment", Buttons: []buttonInfo{
					{Code: "task:create", Name: "创建任务", Path: "/api/v1/tasks", Method: "POST"},
					{Code: "task:cancel", Name: "取消任务", Path: "/api/v1/tasks/*/cancel", Method: "POST"},
					{Code: "task:view", Name: "查看任务", Path: "/api/v1/tasks/*", Method: "GET"},
				}},
				{Name: "邮件配置", Code: "mail-config", Path: "/mail-config", Icon: "MdEmail", Buttons: []buttonInfo{
					{Code: "mail-config:view", Name: "查看邮件配置", Path: "/api/v1/system/mail-config", Method: "GET"},
					{Code: "mail-config:edit", Name: "编辑邮件配置", Path: "/api/v1/system/mail-config", Method: "PUT"},
					{Code: "mail-config:test-smtp", Name: "测试SMTP", Path: "/api/v1/system/mail-config/test-smtp", Method: "POST"},
					{Code: "mail-config:test-imap", Name: "测试IMAP", Path: "/api/v1/system/mail-config/test-imap", Method: "POST"},
					{Code: "mail-config:sync-imap", Name: "同步反馈邮件", Path: "/api/v1/system/mail-config/sync-imap", Method: "POST"},
				}},
				{Name: "用户反馈", Code: "feedback", Path: "/feedback", Icon: "MdFeedback", Buttons: []buttonInfo{
					{Code: "feedback:view", Name: "查看反馈", Path: "/api/v1/feedback", Method: "GET"},
					{Code: "feedback:update-status", Name: "更新反馈状态", Path: "/api/v1/feedback/*/status", Method: "PUT"},
				}},
			},
		},
	}
}

// syncPermissions 确保所有定义的权限都存在于数据库中，并分配给 admin 角色。
// 幂等操作：已存在的权限不会重复创建，仅补齐缺失的权限。
// 支持两级菜单结构：父菜单 → 子菜单 → 按钮。
func syncPermissions(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 获取或创建 admin 角色。
		var adminRole model.Role
		if err := tx.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
			return fmt.Errorf("find admin role: %w", err)
		}

		menuList := defaultMenuList()
		var newPerms []model.Permission
		// allDefinedPerms 收集所有定义的权限（包括已存在的），用于最终统一分配。
		var allDefinedPerms []model.Permission

		for i, parent := range menuList {
			// 查找或创建父菜单权限。
			var parentMenu model.Permission
			err := tx.Where("code = ? AND type = ?", parent.Code, "menu").First(&parentMenu).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				parentMenu = model.Permission{
					Name:      parent.Name,
					Code:      parent.Code,
					Path:      parent.Path,
					Icon:      parent.Icon,
					Type:      "menu",
					SortOrder: i + 1,
				}
				if createErr := tx.Create(&parentMenu).Error; createErr != nil {
					return fmt.Errorf("create parent menu %s: %w", parent.Code, createErr)
				}
				newPerms = append(newPerms, parentMenu)
			} else if err != nil {
				return fmt.Errorf("query parent menu %s: %w", parent.Code, err)
			} else {
				// 确保 sort_order 始终正确。
				if parentMenu.SortOrder != i+1 {
					if updateErr := tx.Model(&parentMenu).Update("sort_order", i+1).Error; updateErr != nil {
						return fmt.Errorf("update sort_order for %s: %w", parent.Code, updateErr)
					}
				}
			}
			allDefinedPerms = append(allDefinedPerms, parentMenu)

			// 处理父菜单自身的按钮权限（如仪表盘）。
			for _, btn := range parent.Buttons {
				var btnPerm model.Permission
				btnErr := tx.Where("code = ? AND type = ?", btn.Code, "button").First(&btnPerm).Error
				if errors.Is(btnErr, gorm.ErrRecordNotFound) {
					btnPerm = model.Permission{
						Name:      btn.Name,
						Code:      btn.Code,
						Path:      btn.Path,
						Method:    btn.Method,
						Type:      "button",
						ParentID:  &parentMenu.ID,
						SortOrder: 0,
					}
					if createErr := tx.Create(&btnPerm).Error; createErr != nil {
						return fmt.Errorf("create button %s: %w", btn.Code, createErr)
					}
					newPerms = append(newPerms, btnPerm)
				} else if btnErr != nil {
					return fmt.Errorf("query button %s: %w", btn.Code, btnErr)
				}
				allDefinedPerms = append(allDefinedPerms, btnPerm)
			}

			if parent.Children == nil {
				continue
			}

			// 处理子菜单。
			for j, child := range parent.Children {
				var childMenu model.Permission
				err := tx.Where("code = ? AND type = ?", child.Code, "menu").First(&childMenu).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					childMenu = model.Permission{
						Name:      child.Name,
						Code:      child.Code,
						Path:      child.Path,
						Icon:      child.Icon,
						Type:      "menu",
						ParentID:  &parentMenu.ID,
						SortOrder: j + 1,
					}
					if createErr := tx.Create(&childMenu).Error; createErr != nil {
						return fmt.Errorf("create child menu %s: %w", child.Code, createErr)
					}
					newPerms = append(newPerms, childMenu)
				} else if err != nil {
					return fmt.Errorf("query child menu %s: %w", child.Code, err)
				} else {
					// 确保 ParentID 和 sort_order 始终正确。
					needUpdate := false
					updates := map[string]interface{}{}
					if childMenu.ParentID == nil || *childMenu.ParentID != parentMenu.ID {
						updates["parent_id"] = parentMenu.ID
						needUpdate = true
					}
					if childMenu.SortOrder != j+1 {
						updates["sort_order"] = j + 1
						needUpdate = true
					}
					if needUpdate {
						if updateErr := tx.Model(&childMenu).Updates(updates).Error; updateErr != nil {
							return fmt.Errorf("update child menu %s: %w", child.Code, updateErr)
						}
					}
				}
				allDefinedPerms = append(allDefinedPerms, childMenu)

				// 查找或创建子按钮权限。
				for _, btn := range child.Buttons {
					var btnPerm model.Permission
					btnErr := tx.Where("code = ? AND type = ?", btn.Code, "button").First(&btnPerm).Error
					if errors.Is(btnErr, gorm.ErrRecordNotFound) {
						btnPerm = model.Permission{
							Name:      btn.Name,
							Code:      btn.Code,
							Path:      btn.Path,
							Method:    btn.Method,
							Type:      "button",
							ParentID:  &childMenu.ID,
							SortOrder: 0,
						}
						if createErr := tx.Create(&btnPerm).Error; createErr != nil {
							return fmt.Errorf("create button %s: %w", btn.Code, createErr)
						}
						newPerms = append(newPerms, btnPerm)
					} else if btnErr != nil {
						return fmt.Errorf("query button %s: %w", btn.Code, btnErr)
					}
					allDefinedPerms = append(allDefinedPerms, btnPerm)
				}
			}
		}

		// 将新增的权限分配给 admin 角色。
		if len(newPerms) > 0 {
			if err := tx.Model(&adminRole).Association("Permissions").Append(newPerms); err != nil {
				return fmt.Errorf("assign new permissions to admin: %w", err)
			}
			log.Printf("Synced %d new permissions to admin role", len(newPerms))
		}

		// 确保所有已定义的权限都分配给 admin 角色（修复历史数据中缺失的绑定）。
		var assignedPerms []model.Permission
		if err := tx.Model(&adminRole).Association("Permissions").Find(&assignedPerms); err != nil {
			return fmt.Errorf("find assigned permissions: %w", err)
		}
		assignedSet := make(map[string]bool, len(assignedPerms))
		for _, p := range assignedPerms {
			assignedSet[p.ID] = true
		}
		var missingPerms []model.Permission
		for _, perm := range allDefinedPerms {
			if !assignedSet[perm.ID] {
				missingPerms = append(missingPerms, perm)
			}
		}
		if len(missingPerms) > 0 {
			if err := tx.Model(&adminRole).Association("Permissions").Append(missingPerms); err != nil {
				return fmt.Errorf("assign missing permissions to admin: %w", err)
			}
			log.Printf("Assigned %d missing permissions to admin role", len(missingPerms))
		}

		return nil
	})
}
