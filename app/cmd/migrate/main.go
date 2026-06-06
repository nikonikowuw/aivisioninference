// Package main provides the database migration command for niko-admin.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/config"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/cache"
	"github.com/niko-admin/niko-admin/internal/pkg/database"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode)

	db, err := database.New(dsn, cfg.DB.MaxOpenConns, cfg.DB.MaxIdleConns)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	mustExec(db, "CREATE EXTENSION IF NOT EXISTS vector")

	// 手动迁移：device_licenses 表的 algorithms 字段从 text[] 转为 jsonb
	// GORM AutoMigrate 无法自动完成此类型转换，需提前手动执行
	tryExec(db, `DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'device_licenses' AND column_name = 'algorithms' AND data_type = 'ARRAY'
			) THEN
				ALTER TABLE device_licenses ALTER COLUMN algorithms TYPE jsonb USING to_jsonb(algorithms);
			END IF;
		END
	$$`)

	// Auto migrate all models.
	if err := db.AutoMigrate(
		// Scaffold models (unchanged).
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.AuditLog{},
		&model.File{},
		&model.FileChunk{},
		&model.Task{},
		&model.BrandConfig{},
		&model.MailConfig{},
		&model.EmailToken{},
		&model.InboundEmail{},
		&model.Feedback{},
		&model.UserRole{},
		&model.RolePermission{},

		// AIVisionInference: Person Management.
		&model.PersonGroup{},
		&model.Person{},
		&model.PersonGroupMember{},
		&model.PersonTag{},
		&model.PersonTagRelation{},
		&model.PersonEmbedding{},
		&model.ImportTask{},

		// AIVisionInference: Device Management.
		&model.DeviceGroup{},
		&model.Device{},
		&model.DeviceGroupMember{},
		&model.GB28181Device{},
		&model.DiscoveredDevice{},

		// AIVisionInference: Algorithm Package Management.
		&model.CategoryCode{},
		&model.AlgorithmPackage{},
		&model.AlgorithmLabelMap{},

		// AIVisionInference: Inference Task Management.
		&model.InferTask{},
		&model.InferTaskAlgorithm{},
		&model.InferTaskStatusHistory{},

		// AIVisionInference: Smart Records (partition parent table).
		&model.SmartRecord{},

		// AIVisionInference: Media Stream Mapping.
		&model.MediaStream{},

		// AIVisionInference: Storage Spaces.
		&model.StorageSpace{},
		&model.StorageCleanupLog{},

		// AIVisionInference: Webhook.
		&model.WebhookConfig{},
		&model.WebhookPushLog{},

		// AIVisionInference: System Management.
		&model.SystemConfig{},
		&model.NetworkInterfaceConfig{},
		&model.TimeConfig{},

		// AIVisionInference: System Config & Async Tasks.
		&model.AISystemConfig{},
		&model.AIAsyncTask{},
		&model.AIVisionTask{},
		&model.AITimeSchedule{},

		// AIVisionInference: Device License (MVP+).
		&model.DeviceLicense{},
	); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// Use tryExec instead of mustExec because smart_records may not be
	// a partitioned table if it was created by GORM AutoMigrate.
	for _, sql := range []string{
		"CREATE TABLE IF NOT EXISTS smart_records_202606 PARTITION OF smart_records FOR VALUES FROM ('2026-06-01') TO ('2026-07-01')",
		"CREATE TABLE IF NOT EXISTS smart_records_202607 PARTITION OF smart_records FOR VALUES FROM ('2026-07-01') TO ('2026-08-01')",
		"CREATE TABLE IF NOT EXISTS smart_records_default PARTITION OF smart_records FOR VALUES FROM (MINVALUE) TO ('2026-06-01')",
	} {
		tryExec(db, sql)
	}

	mustExec(db, "CREATE INDEX IF NOT EXISTS idx_person_embeddings_vector ON person_embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)")

	mustExec(db, "CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_root ON users (is_root) WHERE is_root = true")
	mustExec(db, rootUsernameConstraintSQL())
	mustExec(db, ensureAuditLogSummaryColumnsSQL())
	mustExec(db, dropAuditLogActionColumnSQL())
	mustExec(db, migrateAuditLogResultSummarySQL())
	mustExec(db, addAuditLogActionTypeColumnSQL())
	mustExec(db, addFileMD5ColumnSQL())
	mustExec(db, addGB28181ChannelCountColumnSQL())
	mustExec(db, addGB28181LastCatalogAtColumnSQL())

	// Migrate existing menus to multi-level structure.
	if err := migrateMultiLevelMenu(db); err != nil {
		log.Fatalf("migrate multi-level menu: %v", err)
	}

	// Seed default data.
	if err := seedData(db, cfg.Seed, cfg.Redis); err != nil {
		log.Fatalf("seed data: %v", err)
	}

	log.Println("Migration completed successfully")
}

// mustExec 执行 SQL 语句，失败时终止程序。
func mustExec(db *gorm.DB, sql string) {
	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("migrate: %s: %v", sql[:min(len(sql), 60)], err)
	}
}

// tryExec 执行 SQL 语句，失败时仅记录警告。
func tryExec(db *gorm.DB, sql string) {
	if err := db.Exec(sql).Error; err != nil {
		log.Printf("WARNING: %s: %v", sql[:min(len(sql), 60)], err)
	}
}

// addFileMD5ColumnSQL 返回文件 MD5 字段及索引的幂等迁移语句。
func addFileMD5ColumnSQL() string {
	return `
	ALTER TABLE files ADD COLUMN IF NOT EXISTS md5 varchar(64);
	CREATE INDEX IF NOT EXISTS idx_files_md5 ON files (md5);`
}

// addGB28181ChannelCountColumnSQL 添加 GB28181 设备通道数字段。
func addGB28181ChannelCountColumnSQL() string {
	return `ALTER TABLE gb28181_devices ADD COLUMN IF NOT EXISTS channel_count integer NOT NULL DEFAULT 0;`
}

// addGB28181LastCatalogAtColumnSQL 添加 GB28181 设备最后目录同步时间字段。
func addGB28181LastCatalogAtColumnSQL() string {
	return `ALTER TABLE gb28181_devices ADD COLUMN IF NOT EXISTS last_catalog_at timestamptz;`
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
			{Code: "license-management", Name: "授权管理", Icon: "MdVpnKey", ChildCodes: []string{"license"}},
			{Code: "system-management", Name: "系统管理", Icon: "MdSettings", ChildCodes: []string{"files", "audit-logs", "tasks", "algorithm-packages"}},
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
func seedData(db *gorm.DB, seedCfg config.SeedConfig, redisCfg config.RedisConfig) error {
	// Ensure root user exists.
	var rootCount int64
	if err := db.Model(&model.User{}).Where("username = ?", "root").Count(&rootCount).Error; err != nil {
		return fmt.Errorf("count root user: %w", err)
	}
	if rootCount == 0 {
		if err := db.Transaction(func(tx *gorm.DB) error {
			rootPwd, err := hash.Hash(seedCfg.RootPassword)
			if err != nil {
				return fmt.Errorf("hash root password: %w", err)
			}
			root := model.User{Username: "root", Password: rootPwd, Email: seedCfg.RootEmail, DisplayName: "超级管理员", Status: 1, IsRoot: true}
			if err := tx.Create(&root).Error; err != nil {
				return fmt.Errorf("create root: %w", err)
			}

			// Find admin role if it already exists and assign to root
			var role model.Role
			if err := tx.Where("name = ?", "admin").First(&role).Error; err == nil {
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
	// 用户名和邮箱均不存在时创建默认管理员
	if usernameCount == 0 && emailCount == 0 {
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
			var role model.Role
			if err := tx.Where("name = ?", "admin").First(&role).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				role = model.Role{
					Name:        "admin",
					Description: "系统管理员",
					SortOrder:   1,
					Status:      1,
					Level:       1,
				}
				if err := tx.Create(&role).Error; err != nil {
					return fmt.Errorf("create admin role: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("find admin role: %w", err)
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

	// Invalidate menu tree cache in Redis.
	if rdb, err := cache.New(redisCfg.Host, redisCfg.Port, redisCfg.Password, redisCfg.DB); err == nil {
		defer rdb.Close()
		ctx := context.Background()
		iter := rdb.Scan(ctx, 0, "perm:menu_tree:*", 0).Iterator()
		var count int
		for iter.Next(ctx) {
			if delErr := rdb.Del(ctx, iter.Val()).Err(); delErr == nil {
				count++
			}
		}
		if count > 0 {
			log.Printf("Invalidated %d menu tree cache entries in Redis", count)
		}
	} else {
		log.Printf("Redis not available, skip cache invalidation: %v", err)
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
			Name: "设备管理", Code: "device-management", Path: "/device-management", Icon: "MdVideocam",
			Children: []childMenuDef{
				{Name: "设备列表", Code: "devices", Path: "/devices", Icon: "MdVideocam", Buttons: []buttonInfo{
					{Code: "device:list", Name: "设备列表", Path: "/api/v1/devices", Method: "GET"},
					{Code: "device:create", Name: "创建设备", Path: "/api/v1/devices", Method: "POST"},
					{Code: "device:export", Name: "导出设备", Path: "/api/v1/devices/export", Method: "GET"},
					{Code: "device:import", Name: "导入设备", Path: "/api/v1/devices/import", Method: "POST"},
					{Code: "device:batch-delete", Name: "批量删除设备", Path: "/api/v1/devices/batch-delete", Method: "POST"},
					{Code: "device:edit", Name: "编辑设备", Path: "/api/v1/devices/*", Method: "PUT"},
					{Code: "device:delete", Name: "删除设备", Path: "/api/v1/devices/*", Method: "DELETE"},
					{Code: "device:view", Name: "查看设备", Path: "/api/v1/devices/*", Method: "GET"},
					{Code: "device:test", Name: "测试连接", Path: "/api/v1/devices/*/test", Method: "POST"},
				}},
				{Name: "设备待接入", Code: "device-staging", Path: "/devices/staging", Icon: "MdOutlineDeviceHub", Buttons: []buttonInfo{
					{Code: "device-staging:list", Name: "待接入列表", Path: "/api/v1/device-staging", Method: "GET"},
					{Code: "device-staging:import", Name: "导入设备", Path: "/api/v1/device-staging/*/import", Method: "POST"},
					{Code: "device-staging:ignore", Name: "忽略设备", Path: "/api/v1/device-staging/*/ignore", Method: "POST"},
					{Code: "device-staging:batch-import", Name: "批量导入", Path: "/api/v1/device-staging/batch-import", Method: "POST"},
					{Code: "device-staging:batch-ignore", Name: "批量忽略", Path: "/api/v1/device-staging/batch-ignore", Method: "POST"},
					{Code: "device-staging:scan", Name: "ONVIF扫描", Path: "/api/v1/device-staging/scan-onvif", Method: "POST"},
				}},
				{Name: "设备分组", Code: "device-groups", Path: "/devices/groups", Icon: "MdFolder", Buttons: []buttonInfo{
					{Code: "device-group:list", Name: "分组列表", Path: "/api/v1/device-groups", Method: "GET"},
					{Code: "device-group:create", Name: "创建分组", Path: "/api/v1/device-groups", Method: "POST"},
					{Code: "device-group:edit", Name: "编辑分组", Path: "/api/v1/device-groups/*", Method: "PUT"},
					{Code: "device-group:delete", Name: "删除分组", Path: "/api/v1/device-groups/*", Method: "DELETE"},
					{Code: "device-group:view", Name: "查看分组", Path: "/api/v1/device-groups/*", Method: "GET"},
				}},
			},
		},
		{
			Name: "媒体预览", Code: "media-management", Path: "/media-management", Icon: "MdLiveTv",
			Children: []childMenuDef{
				{Name: "实时预览", Code: "live-view", Path: "/media/live", Icon: "MdViewStream", Buttons: []buttonInfo{
					{Code: "media:play", Name: "获取播放地址", Path: "/api/v1/media/play", Method: "GET"},
					{Code: "media:snapshot", Name: "获取截图", Path: "/api/v1/media/snapshot", Method: "GET"},
				}},
				{Name: "流状态看板", Code: "stream-status", Path: "/media/streams", Icon: "MdTimeline", Buttons: []buttonInfo{
					{Code: "stream:list", Name: "查看流状态", Path: "/api/v1/media/streams", Method: "GET"},
				}},
				{Name: "录像回放", Code: "recordings", Path: "/media/recordings", Icon: "MdVideoLibrary", Buttons: []buttonInfo{
					{Code: "recording:list", Name: "录像列表", Path: "/api/v1/media/recordings", Method: "GET"},
					{Code: "recording:playback", Name: "录像回放", Path: "/api/v1/media/recordings/*/playback", Method: "POST"},
					{Code: "recording:start", Name: "开始录像", Path: "/api/v1/media/recordings/start", Method: "POST"},
					{Code: "recording:stop", Name: "停止录像", Path: "/api/v1/media/recordings/stop", Method: "POST"},
				}},
			},
		},
		{
			Name: "智能记录", Code: "smart-records", Path: "/smart-records", Icon: "MdNotificationsActive",
			Buttons: []buttonInfo{
				{Code: "records:recognition:list", Name: "查看识别记录", Path: "/api/v1/smart-records", Method: "GET"},
				{Code: "records:alarm:list", Name: "查看告警记录", Path: "/api/v1/smart-records", Method: "GET"},
				{Code: "records:capture:list", Name: "查看抓拍记录", Path: "/api/v1/smart-records", Method: "GET"},
				{Code: "records:export", Name: "导出智能记录", Path: "/api/v1/smart-records/export", Method: "GET"},
				{Code: "records:batch-delete", Name: "批量删除智能记录", Path: "/api/v1/smart-records/batch-delete", Method: "POST"},
				{Code: "records:export-selected", Name: "导出选定智能记录", Path: "/api/v1/smart-records/export-selected", Method: "POST"},
				{Code: "records:category-codes:list", Name: "查看类别编码", Path: "/api/v1/smart-records/category-codes", Method: "GET"},
			},
			Children: nil,
		},
		{
			Name: "用户管理", Code: "user-management", Path: "/user-management", Icon: "MdPeople",
			Children: []childMenuDef{
				{Name: "用户", Code: "users", Path: "/users", Icon: "MdPerson", Buttons: []buttonInfo{
					{Code: "user:list", Name: "用户列表", Path: "/api/v1/users", Method: "GET"},
					{Code: "user:create", Name: "创建用户", Path: "/api/v1/users", Method: "POST"},
					{Code: "user:export", Name: "导出用户", Path: "/api/v1/users/export", Method: "GET"},
					{Code: "user:import", Name: "导入用户", Path: "/api/v1/users/import", Method: "POST"},
					{Code: "user:batch-delete", Name: "批量删除用户", Path: "/api/v1/users/batch-delete", Method: "POST"},
					{Code: "user:batch-status", Name: "批量更新用户状态", Path: "/api/v1/users/batch-status", Method: "PUT"},
					{Code: "user:edit", Name: "编辑用户", Path: "/api/v1/users/*", Method: "PUT"},
					{Code: "user:delete", Name: "删除用户", Path: "/api/v1/users/*", Method: "DELETE"},
					{Code: "user:view", Name: "查看用户", Path: "/api/v1/users/*", Method: "GET"},
					{Code: "user:reset-password", Name: "重置密码", Path: "/api/v1/users/*/password", Method: "PUT"},
					{Code: "user:upload-avatar", Name: "上传头像", Path: "/api/v1/users/*/avatar", Method: "POST"},
				}},
				{Name: "角色", Code: "roles", Path: "/roles", Icon: "MdSecurity", Buttons: []buttonInfo{
					{Code: "role:create", Name: "创建角色", Path: "/api/v1/roles", Method: "POST"},
					{Code: "role:export", Name: "导出角色", Path: "/api/v1/roles/export", Method: "GET"},
					{Code: "role:batch-delete", Name: "批量删除角色", Path: "/api/v1/roles/batch-delete", Method: "POST"},
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
			Name: "授权管理", Code: "license-management", Path: "/license-management", Icon: "MdVpnKey",
			Children: []childMenuDef{
				{Name: "算法授权", Code: "license", Path: "/license", Icon: "MdVpnKey", Buttons: []buttonInfo{
					{Code: "license:fingerprint", Name: "查看设备指纹", Path: "/api/v1/license/fingerprint", Method: "GET"},
					{Code: "license:upload", Name: "上传授权文件", Path: "/api/v1/license/upload", Method: "POST"},
					{Code: "license:active", Name: "查看当前授权", Path: "/api/v1/license/active", Method: "GET"},
					{Code: "license:list", Name: "授权列表", Path: "/api/v1/license", Method: "GET"},
					{Code: "license:check", Name: "校验算法授权", Path: "/api/v1/license/check", Method: "GET"},
				}},
			},
		},
		{
			Name: "AI视觉推理", Code: "aivision", Path: "/aivision", Icon: "MdRemoveRedEye",
			Children: []childMenuDef{
				{Name: "时间配置", Code: "ai-time-schedules", Path: "/ai-time-schedules", Icon: "MdTimeline", Buttons: []buttonInfo{
					{Code: "ai-time-schedules:list", Name: "时间配置列表", Path: "/api/v1/ai-time-schedules", Method: "GET"},
					{Code: "ai-time-schedules:create", Name: "创建时间配置", Path: "/api/v1/ai-time-schedules", Method: "POST"},
					{Code: "ai-time-schedules:edit", Name: "编辑时间配置", Path: "/api/v1/ai-time-schedules/*", Method: "PUT"},
					{Code: "ai-time-schedules:delete", Name: "删除时间配置", Path: "/api/v1/ai-time-schedules/*", Method: "DELETE"},
				}},
				{Name: "推理任务", Code: "aivisiontasks", Path: "/ai-tasks", Icon: "MdAssignment", Buttons: []buttonInfo{
					{Code: "aivisiontasks:list", Name: "任务列表", Path: "/api/v1/aivisiontasks", Method: "GET"},
					{Code: "aivisiontasks:create", Name: "创建任务", Path: "/api/v1/aivisiontasks", Method: "POST"},
					{Code: "aivisiontasks:edit", Name: "编辑任务", Path: "/api/v1/aivisiontasks/*", Method: "PUT"},
					{Code: "aivisiontasks:delete", Name: "删除任务", Path: "/api/v1/aivisiontasks/*", Method: "DELETE"},
					{Code: "aivisiontasks:check-conflict", Name: "检查冲突", Path: "/api/v1/aivisiontasks/check-conflict", Method: "POST"},
				}},
			},
		},
		{
			Name: "系统管理", Code: "system-management", Path: "/system-management", Icon: "MdSettings",
			Children: []childMenuDef{
				{Name: "文件", Code: "files", Path: "/files", Icon: "MdFolder", Buttons: []buttonInfo{
					{Code: "file:list", Name: "文件列表", Path: "/api/v1/files", Method: "GET"},
					{Code: "file:export", Name: "导出文件", Path: "/api/v1/files/export", Method: "GET"},
					{Code: "file:batch-delete", Name: "批量删除文件", Path: "/api/v1/files/batch-delete", Method: "POST"},
					{Code: "file:upload", Name: "上传文件", Path: "/api/v1/files/upload/**", Method: "POST"},
					{Code: "file:check", Name: "校验文件", Path: "/api/v1/files/upload/check", Method: "POST"},
					{Code: "file:upload-progress", Name: "上传进度", Path: "/api/v1/files/upload/*/progress", Method: "GET"},
					{Code: "file:delete", Name: "删除文件", Path: "/api/v1/files/*", Method: "DELETE"},
					{Code: "file:view", Name: "查看文件", Path: "/api/v1/files/*", Method: "GET"},
					{Code: "file:download", Name: "下载文件", Path: "/api/v1/files/*/download", Method: "GET"},
				}},
				{Name: "算法包管理", Code: "algorithm-packages", Path: "/algorithm-packages", Icon: "MdExtension", Buttons: []buttonInfo{
					{Code: "algorithm-package:list", Name: "算法包列表", Path: "/api/v1/algorithmpackages", Method: "GET"},
					{Code: "algorithm-package:upload", Name: "上传算法包", Path: "/api/v1/algorithmpackages/upload", Method: "POST"},
					{Code: "algorithm-package:view", Name: "查看算法包", Path: "/api/v1/algorithmpackages/*", Method: "GET"},
					{Code: "algorithm-package:delete", Name: "删除算法包", Path: "/api/v1/algorithmpackages/*", Method: "DELETE"},
				}},
				{Name: "审计日志", Code: "audit-logs", Path: "/audit-logs", Icon: "MdHistory", Buttons: []buttonInfo{
					{Code: "audit:view", Name: "查看审计日志", Path: "/api/v1/audit-logs", Method: "GET"},
					{Code: "audit:export", Name: "导出审计日志", Path: "/api/v1/audit-logs/export", Method: "GET"},
				}},
				{Name: "任务", Code: "tasks", Path: "/tasks", Icon: "MdAssignment", Buttons: []buttonInfo{
					{Code: "task:list", Name: "任务列表", Path: "/api/v1/tasks", Method: "GET"},
					{Code: "task:create", Name: "创建任务", Path: "/api/v1/tasks", Method: "POST"},
					{Code: "task:export", Name: "导出任务", Path: "/api/v1/tasks/export", Method: "GET"},
					{Code: "task:batch-cancel", Name: "批量取消任务", Path: "/api/v1/tasks/batch-cancel", Method: "POST"},
					{Code: "task:cancel", Name: "取消任务", Path: "/api/v1/tasks/*/cancel", Method: "POST"},
					{Code: "task:view", Name: "查看任务", Path: "/api/v1/tasks/*", Method: "GET"},
				}},
				{Name: "品牌配置", Code: "brand-config", Path: "/brand-config", Icon: "MdPalette", Buttons: []buttonInfo{
					{Code: "brand-config:view", Name: "查看品牌配置", Path: "/api/v1/system/brand-config", Method: "GET"},
					{Code: "brand-config:edit", Name: "编辑品牌配置", Path: "/api/v1/system/brand-config", Method: "PUT"},
					{Code: "brand-config:upload-logo", Name: "上传品牌Logo", Path: "/api/v1/system/brand-config/logo", Method: "POST"},
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
					{Code: "feedback:export", Name: "导出反馈", Path: "/api/v1/feedback/export", Method: "GET"},
					{Code: "feedback:batch-status", Name: "批量更新反馈状态", Path: "/api/v1/feedback/batch-status", Method: "PUT"},
					{Code: "feedback:update-status", Name: "更新反馈状态", Path: "/api/v1/feedback/*/status", Method: "PUT"},
				}},
				{Name: "系统配置", Code: "system-config", Path: "/system/config", Icon: "MdSettings", Buttons: []buttonInfo{
					// 运行状态 Tab
					{Code: "system:status:view", Name: "查看运行状态", Path: "/api/v1/system/status/realtime", Method: "GET"},
					{Code: "system:status:resources", Name: "查看资源状态", Path: "/api/v1/system/status/resources", Method: "GET"},
					{Code: "system:status:services", Name: "查看服务状态", Path: "/api/v1/system/status/services", Method: "GET"},
					// 网络配置 Tab
					{Code: "system:network:view", Name: "查看网络配置", Path: "/api/v1/system/network", Method: "GET"},
					{Code: "system:network:apply", Name: "应用网络配置", Path: "/api/v1/system/network/apply", Method: "POST"},
					{Code: "system:network:confirm", Name: "确认网络配置", Path: "/api/v1/system/network/confirm", Method: "POST"},
					{Code: "system:network:rollback", Name: "回滚网络配置", Path: "/api/v1/system/network/rollback", Method: "POST"},
					// 时间配置 Tab
					{Code: "system:time:view", Name: "查看时间配置", Path: "/api/v1/system/time", Method: "GET"},
					{Code: "system:time:manual", Name: "手动设置时间", Path: "/api/v1/system/time/manual", Method: "POST"},
					{Code: "system:time:timezone", Name: "设置时区", Path: "/api/v1/system/time/timezone", Method: "PUT"},
					{Code: "system:ntp:view", Name: "查看NTP配置", Path: "/api/v1/system/time/ntp", Method: "GET"},
					{Code: "system:ntp:manage", Name: "管理NTP服务器", Path: "/api/v1/system/time/ntp/servers", Method: "POST"},
					{Code: "system:ntp:sync", Name: "同步NTP时间", Path: "/api/v1/system/time/ntp/sync", Method: "POST"},
					// 告警上报 Tab
					{Code: "system:webhook:list", Name: "Webhook列表", Path: "/api/v1/system/webhook", Method: "GET"},
					{Code: "system:webhook:create", Name: "创建Webhook", Path: "/api/v1/system/webhook", Method: "POST"},
					{Code: "system:webhook:edit", Name: "编辑Webhook", Path: "/api/v1/system/webhook/*", Method: "PUT"},
					{Code: "system:webhook:delete", Name: "删除Webhook", Path: "/api/v1/system/webhook/*", Method: "DELETE"},
					{Code: "system:webhook:test", Name: "测试Webhook", Path: "/api/v1/system/webhook/*/test", Method: "POST"},
					{Code: "system:webhook:logs", Name: "查看推送日志", Path: "/api/v1/system/webhook/logs", Method: "GET"},
					// 存储配置 Tab
					{Code: "system:storage:view", Name: "查看存储配置", Path: "/api/v1/system/storage/config", Method: "GET"},
					{Code: "system:storage:edit", Name: "编辑存储配置", Path: "/api/v1/system/storage/config", Method: "PUT"},
					{Code: "system:storage:cleanup-logs", Name: "查看清理日志", Path: "/api/v1/system/storage/cleanup-logs", Method: "GET"},
					{Code: "system:storage:cleanup", Name: "手动触发清理", Path: "/api/v1/system/storage/cleanup/run", Method: "POST"},
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
