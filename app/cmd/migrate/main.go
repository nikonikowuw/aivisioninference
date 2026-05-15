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
	seedData(db)

	log.Println("Migration completed successfully")
}

// seedData inserts the default admin user, role, and permissions if they do not exist.
func seedData(db *gorm.DB) {
	// Check if admin user exists.
	var count int64
	db.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	// Create admin user.
	hashedPwd, err := hash.Hash("admin123")
	if err != nil {
		log.Printf("hash password: %v", err)
		return
	}

	admin := model.User{
		Username:    "admin",
		Password:    hashedPwd,
		Email:       "admin@example.com",
		DisplayName: "管理员",
		Status:      1,
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Printf("create admin: %v", err)
		return
	}

	// Create admin role.
	role := model.Role{
		Name:        "admin",
		Description: "系统管理员",
		SortOrder:   1,
		Status:      1,
	}
	if err := db.Create(&role).Error; err != nil {
		log.Printf("create admin role: %v", err)
		return
	}

	// Assign admin role to admin user.
	if err := db.Model(&admin).Association("Roles").Append(&role); err != nil {
		log.Printf("assign admin role: %v", err)
		return
	}

	// Create default permissions.
	perms := []model.Permission{
		{Name: "用户管理", Code: "user:list", Path: "/api/v1/users", Method: "GET", Type: "api"},
		{Name: "创建用户", Code: "user:create", Path: "/api/v1/users", Method: "POST", Type: "api"},
		{Name: "编辑用户", Code: "user:update", Path: "/api/v1/users/*", Method: "PUT", Type: "api"},
		{Name: "删除用户", Code: "user:delete", Path: "/api/v1/users/*", Method: "DELETE", Type: "api"},
		{Name: "角色管理", Code: "role:list", Path: "/api/v1/roles", Method: "GET", Type: "api"},
		{Name: "创建角色", Code: "role:create", Path: "/api/v1/roles", Method: "POST", Type: "api"},
		{Name: "编辑角色", Code: "role:update", Path: "/api/v1/roles/*", Method: "PUT", Type: "api"},
		{Name: "删除角色", Code: "role:delete", Path: "/api/v1/roles/*", Method: "DELETE", Type: "api"},
		{Name: "审计日志", Code: "audit:list", Path: "/api/v1/audit-logs", Method: "GET", Type: "api"},
		{Name: "文件管理", Code: "file:list", Path: "/api/v1/files", Method: "GET", Type: "api"},
		{Name: "任务管理", Code: "task:list", Path: "/api/v1/tasks", Method: "GET", Type: "api"},
	}

	for i := range perms {
		if err := db.Create(&perms[i]).Error; err != nil {
			log.Printf("create permission %s: %v", perms[i].Code, err)
			continue
		}
	}

	// Assign all permissions to admin role.
	if err := db.Model(&role).Association("Permissions").Append(perms); err != nil {
		log.Printf("assign permissions: %v", err)
		return
	}

	log.Println("Default data seeded: admin user, admin role, default permissions")
}
