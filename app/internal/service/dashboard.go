// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/repository"
)

// DashboardStats 包含仪表盘统计信息的数据结构
type DashboardStats struct {
	TotalUsers  int64          `json:"total_users"`  // 总用户数
	TotalRoles  int64          `json:"total_roles"`  // 总角色数
	TotalFiles  int64          `json:"total_files"`  // 总文件数
	TotalTasks  int64          `json:"total_tasks"`  // 总任务数
	ActiveTasks int64          `json:"active_tasks"` // 激活态的任务数
	UserStats   []UserStatData `json:"user_stats"`   // 用户统计数据
	AuditLogs   []AuditLogData `json:"audit_logs"`   // 最近审计日志
}

// UserStatData 表示前端折线图使用的单日用户统计数据。
type UserStatData struct {
	Date   string `json:"date"`   // 日期标签，格式为 MM-DD
	New    int64  `json:"new"`    // 当日新增用户数
	Active int64  `json:"active"` // 当日活跃用户数
}

// AuditLogData 表示仪表盘最近活动表格中的审计日志摘要。
type AuditLogData struct {
	Username  string `json:"username"`   // 操作用户名称
	Action    string `json:"action"`     // 操作类型
	Method    string `json:"method"`     // HTTP 请求方法
	CreatedAt string `json:"created_at"` // 创建时间字符串
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
func (s *DashboardService) Stats(ctx context.Context, lang string) (*DashboardStats, error) {
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

	userStats, err := s.dashRepo.RecentUserStats(ctx, 7)
	if err != nil {
		return nil, err
	}
	stats.UserStats = make([]UserStatData, len(userStats))
	for i, row := range userStats {
		stats.UserStats[i] = UserStatData{
			Date:   row.Date,
			New:    row.New,
			Active: row.Active,
		}
	}

	logs, err := s.dashRepo.RecentAuditLogs(ctx, 5)
	if err != nil {
		return nil, err
	}
	stats.AuditLogs = make([]AuditLogData, len(logs))
	for i, log := range logs {
		stats.AuditLogs[i] = AuditLogData{
			Username:  log.Username,
			Action:    log.ActionType,
			Method:    log.RequestMethod,
			CreatedAt: log.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return stats, nil
}
