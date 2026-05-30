// Package repository 提供数据访问层实现，封装 GORM 数据库操作。
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// DashboardRepository 处理仪表盘统计相关的数据库查询操作
type DashboardRepository struct {
	db *gorm.DB // GORM 数据库连接实例
}

// NewDashboardRepository 创建并返回一个新的 DashboardRepository 实例
func NewDashboardRepository(db *gorm.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// CountUsers 统计系统中的总用户数
func (r *DashboardRepository) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error
	return count, err
}

// CountRoles 统计系统中的总角色数
func (r *DashboardRepository) CountRoles(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Role{}).Count(&count).Error
	return count, err
}

// CountFiles 统计系统中的总上传文件数
func (r *DashboardRepository) CountFiles(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.File{}).Count(&count).Error
	return count, err
}

// CountTasks 统计异步后台任务的总数
func (r *DashboardRepository) CountTasks(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Task{}).Count(&count).Error
	return count, err
}

// CountActiveTasks 统计当前处于激活状态（pending 等待中 或 running 执行中）的异步任务数
func (r *DashboardRepository) CountActiveTasks(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Task{}).
		Where("status IN ?", []string{"pending", "running"}).
		Count(&count).Error
	return count, err
}

// RecentAuditLogs 查询最近的审计日志列表。
func (r *DashboardRepository) RecentAuditLogs(ctx context.Context, limit int) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// UserStatRow 表示某一天的新增用户和活跃用户统计行。
type UserStatRow struct {
	Date   string // 日期标签，格式为 MM-DD
	New    int64  // 当日新增用户数
	Active int64  // 当日活跃用户数
}

// RecentUserStats 查询最近 days 天的用户新增和活跃统计。
func (r *DashboardRepository) RecentUserStats(ctx context.Context, days int) ([]UserStatRow, error) {
	stats := make([]UserStatRow, days)
	now := time.Now()
	for i := 0; i < days; i++ {
		day := now.AddDate(0, 0, i-days+1)
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
		end := start.AddDate(0, 0, 1)

		newUsers, err := r.countNewUsers(ctx, start, end)
		if err != nil {
			return nil, err
		}
		activeUsers, err := r.countActiveUsers(ctx, start, end)
		if err != nil {
			return nil, err
		}

		stats[i] = UserStatRow{
			Date:   day.Format("01-02"),
			New:    newUsers,
			Active: activeUsers,
		}
	}
	return stats, nil
}

// countNewUsers 统计指定日期范围内新增用户数。
func (r *DashboardRepository) countNewUsers(ctx context.Context, start, end time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&count).Error
	return count, err
}

// countActiveUsers 统计指定日期范围内有审计记录的去重用户数。
func (r *DashboardRepository) countActiveUsers(ctx context.Context, start, end time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AuditLog{}).
		Where("created_at >= ? AND created_at < ? AND user_id IS NOT NULL", start, end).
		Distinct("user_id").
		Count(&count).Error
	return count, err
}
