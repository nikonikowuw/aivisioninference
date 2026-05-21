package service

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/repository"
)

// DashboardStats 包含仪表盘统计信息的数据结构
type DashboardStats struct {
	TotalUsers  int64 `json:"total_users"`  // 总用户数
	TotalRoles  int64 `json:"total_roles"`  // 总角色数
	TotalFiles  int64 `json:"total_files"`  // 总文件数
	TotalTasks  int64 `json:"total_tasks"`  // 总任务数
	ActiveTasks int64 `json:"active_tasks"` // 激活态的任务数
}

// DashboardService 处理仪表盘相关的业务逻辑
type DashboardService struct {
	dashRepo *repository.DashboardRepository // 仪表盘数据持久化接口
}

// NewDashboardService 创建并返回一个新的 DashboardService 实例
func NewDashboardService(dashRepo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{dashRepo: dashRepo}
}

// Stats 聚合并返回系统的统计数据
func (s *DashboardService) Stats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

	users, err := s.dashRepo.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = users

	roles, err := s.dashRepo.CountRoles(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalRoles = roles

	files, err := s.dashRepo.CountFiles(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalFiles = files

	tasks, err := s.dashRepo.CountTasks(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalTasks = tasks

	active, err := s.dashRepo.CountActiveTasks(ctx)
	if err != nil {
		return nil, err
	}
	stats.ActiveTasks = active

	return stats, nil
}
