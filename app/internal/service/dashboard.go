package service

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/repository"
)

// DashboardStats holds aggregated statistics for the dashboard.
type DashboardStats struct {
	TotalUsers  int64 `json:"total_users"`
	TotalRoles  int64 `json:"total_roles"`
	TotalFiles  int64 `json:"total_files"`
	TotalTasks  int64 `json:"total_tasks"`
	ActiveTasks int64 `json:"active_tasks"`
}

// DashboardService handles business logic for dashboard operations.
type DashboardService struct {
	dashRepo *repository.DashboardRepository
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(dashRepo *repository.DashboardRepository) *DashboardService {
	return &DashboardService{dashRepo: dashRepo}
}

// Stats returns aggregated system statistics.
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
