package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// DashboardRepository handles database operations for dashboard statistics.
type DashboardRepository struct {
	db *gorm.DB
}

// NewDashboardRepository creates a new DashboardRepository.
func NewDashboardRepository(db *gorm.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// CountUsers returns the total number of users.
func (r *DashboardRepository) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error
	return count, err
}

// CountRoles returns the total number of roles.
func (r *DashboardRepository) CountRoles(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Role{}).Count(&count).Error
	return count, err
}

// CountFiles returns the total number of files.
func (r *DashboardRepository) CountFiles(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.File{}).Count(&count).Error
	return count, err
}

// CountTasks returns the total number of tasks.
func (r *DashboardRepository) CountTasks(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Task{}).Count(&count).Error
	return count, err
}

// CountActiveTasks returns the number of pending or running tasks.
func (r *DashboardRepository) CountActiveTasks(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Task{}).
		Where("status IN ?", []string{"pending", "running"}).
		Count(&count).Error
	return count, err
}
