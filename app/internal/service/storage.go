// Package service 提供业务逻辑层
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"golang.org/x/sys/unix"
	"gorm.io/gorm"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/model"
)

// CleanupMode 存储清理策略
// cron: CRON 定时清理、threshold: 阈值清理
type CleanupMode string

const (
	CleanupModeCron      CleanupMode = "cron"
	CleanupModeThreshold CleanupMode = "threshold"
)

// StorageConfig 存储配置
// 注意: 两种模式互斥，同一时间只有一种生效
type StorageConfig struct {
	Enabled            bool        `json:"enabled"`
	CleanupMode        CleanupMode `json:"cleanup_mode"`
	RetentionDays      int         `json:"retention_days"`
	CronExpression     string      `json:"cron_expression"`
	ThresholdValue     float64     `json:"threshold_value"`
	TargetPercentage   float64     `json:"target_percentage"`
	ThresholdCheckCron string      `json:"threshold_check_cron"`
}

func (c *StorageConfig) validate() error {
	if c.CleanupMode != CleanupModeCron && c.CleanupMode != CleanupModeThreshold {
		return fmt.Errorf("cleanup_mode must be 'cron' or 'threshold'")
	}
	if c.CleanupMode == CleanupModeThreshold {
		if c.TargetPercentage >= c.ThresholdValue {
			return fmt.Errorf("target_percentage must be less than threshold_value")
		}
		if c.ThresholdCheckCron == "" {
			return fmt.Errorf("threshold_check_cron is required for threshold mode")
		}
		if _, err := cron.ParseStandard(c.ThresholdCheckCron); err != nil {
			return fmt.Errorf("invalid threshold_check_cron: %w", err)
		}
	}
	if c.CleanupMode == CleanupModeCron {
		if c.CronExpression == "" {
			return fmt.Errorf("cron_expression is required for cron mode")
		}
		if _, err := cron.ParseStandard(c.CronExpression); err != nil {
			return fmt.Errorf("invalid cron_expression: %w", err)
		}
	}
	return nil
}

// CleanupLogResponse 清理日志响应
type CleanupLogResponse struct {
	ID             string   `json:"id"`
	SpaceID        string   `json:"space_id"`
	SpaceName      string   `json:"space_name"`
	DiskUsageBefore int64   `json:"disk_usage_before"`
	DiskUsageAfter  int64   `json:"disk_usage_after"`
	CleanedRecords int      `json:"cleaned_records"`
	CleanedFiles   int      `json:"cleaned_files"`
	FreedSpace     int64    `json:"freed_space"`
	Status         string   `json:"status"`
	ErrorMessage   string   `json:"error_message,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// StorageService 存储配置服务
type StorageService struct {
	db    *gorm.DB
	rdb   *redis.Client
}

// NewStorageService 创建存储配置服务
func NewStorageService(db *gorm.DB, rdb *redis.Client) *StorageService {
	return &StorageService{
		db:  db,
		rdb: rdb,
	}
}

func defaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		Enabled:            true,
		CleanupMode:        CleanupModeCron,
		RetentionDays:      7,
		CronExpression:     "0 2 * * *",
		ThresholdValue:     90,
		TargetPercentage:   70,
		ThresholdCheckCron: "*/5 * * * *",
	}
}

// GetConfig 获取存储配置
func (s *StorageService) GetConfig() (*StorageConfig, error) {
	var space model.StorageSpace
	if err := s.db.First(&space).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return defaultStorageConfig(), nil
		}
		return nil, err
	}

	// 清理模式直接从 model 字段读取
	mode := CleanupMode(space.CleanupMode)
	if mode != CleanupModeCron && mode != CleanupModeThreshold {
		mode = CleanupModeCron
	}

	return &StorageConfig{
		Enabled:            space.Enabled,
		CleanupMode:        mode,
		RetentionDays:      space.RetentionDays,
		CronExpression:     space.CronExpression,
		ThresholdValue:     space.ThresholdValue,
		TargetPercentage:   space.TargetPercentage,
		ThresholdCheckCron: space.ThresholdCheckCron,
	}, nil
}

// UpdateConfig 更新存储配置
func (s *StorageService) UpdateConfig(req *StorageConfig) error {
	if err := req.validate(); err != nil {
		return err
	}

	var space model.StorageSpace
	err := s.db.First(&space).Error
	if err == gorm.ErrRecordNotFound {
		return s.db.Create(&model.StorageSpace{
			SpaceName:          "default",
			SpaceType:          model.SpaceTypeSnapshot,
			StorageType:        model.StorageTypeLocal,
			LocalBasePath:      "/data/storage",
			RetentionDays:      req.RetentionDays,
			CleanupMode:        string(req.CleanupMode),
			ThresholdType:      model.ThresholdTypePercent,
			ThresholdValue:     req.ThresholdValue,
			TargetPercentage:   req.TargetPercentage,
			CronExpression:     req.CronExpression,
			ThresholdCheckCron: req.ThresholdCheckCron,
			Enabled:            req.Enabled,
		}).Error
	}
	if err != nil {
		return err
	}

	space.RetentionDays = req.RetentionDays
	space.CleanupMode = string(req.CleanupMode)
	space.ThresholdValue = req.ThresholdValue
	space.TargetPercentage = req.TargetPercentage
	space.CronExpression = req.CronExpression
	space.ThresholdCheckCron = req.ThresholdCheckCron
	space.Enabled = req.Enabled
	space.ThresholdType = model.ThresholdTypePercent
	return s.db.Save(&space).Error
}

// IsOverThreshold 检查磁盘使用率是否超过配置的阈值
func (s *StorageService) IsOverThreshold() (bool, error) {
	var space model.StorageSpace
	if err := s.db.First(&space).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	if space.ThresholdValue <= 0 {
		return false, nil
	}

	usage, err := diskUsagePercent(space.LocalBasePath)
	if err != nil {
		return false, err
	}
	return usage > space.ThresholdValue, nil
}

// GetLastCleanupLog 获取最近一次清理日志
func (s *StorageService) GetLastCleanupLog() (*CleanupLogResponse, error) {
	var log model.StorageCleanupLog
	if err := s.db.Order("created_at DESC").First(&log).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &CleanupLogResponse{
		ID:        log.ID,
		StartedAt: log.StartedAt,
		Status:    log.Status,
	}, nil
}

// ListCleanupLogs 获取清理日志
func (s *StorageService) ListCleanupLogs(page, pageSize int) ([]CleanupLogResponse, int64, error) {
	var logs []model.StorageCleanupLog
	var total int64

	if err := s.db.Model(&model.StorageCleanupLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	result := make([]CleanupLogResponse, 0, len(logs))
	for _, log := range logs {
		result = append(result, CleanupLogResponse{
			ID:              log.ID,
			SpaceID:         log.SpaceID,
			DiskUsageBefore: log.DiskUsageBefore,
			DiskUsageAfter:  log.DiskUsageAfter,
			CleanedRecords:  log.CleanedRecords,
			CleanedFiles:    log.CleanedFiles,
			FreedSpace:      log.FreedSpace,
			Status:          log.Status,
			ErrorMessage:    log.ErrorMessage,
			StartedAt:       log.StartedAt,
			CompletedAt:     log.CompletedAt,
		})
	}

	return result, total, nil
}

// unlockScript 原子性释放锁：仅在锁值匹配时删除，防止误删其他实例的锁
const unlockScript = `if redis.call("get",KEYS[1]) == ARGV[1] then return redis.call("del",KEYS[1]) else return 0 end`

// RunCleanup 执行清理
func (s *StorageService) RunCleanup() error {
	// 获取分布式锁
	lockKey := "storage_cleanup_lock"
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	lockTTL := 10 * time.Minute

	// 尝试获取锁
	locked, err := s.rdb.SetNX(context.Background(), lockKey, lockValue, lockTTL).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %v", err)
	}
	if !locked {
		return apperrors.New(apperrors.ErrCleanupRunning, "")
	}

	// 安全释放锁：仅在锁值匹配时删除，防止误删其他实例的锁
	defer func() {
		s.rdb.Eval(context.Background(), unlockScript, []string{lockKey}, lockValue)
	}()

	// 获取配置
	var space model.StorageSpace
	if err := s.db.First(&space).Error; err != nil {
		return err
	}

	// 创建清理日志
	log := model.StorageCleanupLog{
		SpaceID:   space.ID,
		Status:    model.StorageCleanupStatusRunning,
		StartedAt: time.Now(),
	}
	s.db.Create(&log)

	// 执行清理并记录结果
	cleanupErr := s.executeCleanup(&space, &log)
	now := time.Now()
	log.CompletedAt = &now
	if cleanupErr != nil {
		log.Status = model.StorageCleanupStatusFailed
		log.ErrorMessage = cleanupErr.Error()
	} else {
		log.Status = model.StorageCleanupStatusSuccess
	}
	s.db.Save(&log)

	return cleanupErr
}

// executeCleanup 执行清理逻辑
func (s *StorageService) executeCleanup(space *model.StorageSpace, log *model.StorageCleanupLog) error {
	if !space.Enabled {
		return apperrors.New(apperrors.ErrCleanupDisabled, "")
	}

	// 过期清理：删除超过保留天数的数据
	if space.RetentionDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -space.RetentionDays)
		result := s.db.Where("capture_time < ?", cutoff).Delete(&model.SmartRecord{})
		if result.Error != nil {
			return result.Error
		}
		log.CleanedRecords = int(result.RowsAffected)
	}

	return nil
}

// diskUsagePercent 返回指定路径的磁盘使用率
func diskUsagePercent(path string) (float64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to stat filesystem %s: %v", path, err)
	}
	blockSize := uint64(stat.Bsize)
	totalBytes := stat.Blocks * blockSize
	usedBytes := (stat.Blocks - stat.Bfree) * blockSize
	if totalBytes == 0 {
		return 0, nil
	}
	return float64(usedBytes) / float64(totalBytes) * 100, nil
}
