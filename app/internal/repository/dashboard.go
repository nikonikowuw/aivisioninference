package repository

import (
	"context"

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

