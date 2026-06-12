// 临时工具：修复算法包的 so_path 字段
package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

func main() {
	// 从环境变量读取 DSN
	dsn := os.Getenv("NIKO_DB_DSN")
	if dsn == "" {
		host := os.Getenv("NIKO_DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("NIKO_DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("NIKO_DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("NIKO_DB_PASSWORD")
		if password == "" {
			password = "postgres"
		}
		dbname := os.Getenv("NIKO_DB_NAME")
		if dbname == "" {
			dbname = "niko_admin"
		}
		sslmode := os.Getenv("NIKO_DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	soPath := "/Users/niko/dev/go/aivisioninference/algorithms/M/face_recognition/1.0.0/nikoniko_detector.so"

	// 检查文件是否存在
	if _, err := os.Stat(soPath); err != nil {
		log.Fatalf("✗ .so 文件不存在: %s\n%v", soPath, err)
	}

	// 更新所有 face_recognition 1.0.0 passed 的包
	result := db.Model(&model.AlgorithmPackage{}).
		Where("algorithm_name = ? AND version = ? AND self_check_status = ?", "face_recognition", "1.0.0", model.SelfCheckStatusPassed).
		Update("so_path", soPath)

	if result.Error != nil {
		log.Fatalf("更新失败: %v", result.Error)
	}

	fmt.Printf("✓ 已更新 %d 条算法包记录\n", result.RowsAffected)
	fmt.Printf("✓ so_path → %s\n", soPath)
}
