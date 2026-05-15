package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// DashboardHandler handles dashboard statistics HTTP requests.
type DashboardHandler struct {
	db *gorm.DB
}

// NewDashboardHandler creates a new DashboardHandler with the given database.
func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// dashboardStats holds aggregated statistics for the dashboard.
type dashboardStats struct {
	TotalUsers  int64 `json:"total_users"`
	TotalRoles  int64 `json:"total_roles"`
	TotalFiles  int64 `json:"total_files"`
	TotalTasks  int64 `json:"total_tasks"`
	ActiveTasks int64 `json:"active_tasks"`
}

// Stats returns dashboard statistics (total users, roles, files, tasks).
//
// @Summary      仪表盘统计
// @Description  返回系统概览统计数据
// @Tags         仪表盘
// @Produce      json
// @Success      200  {object}  dto.Response{data=handler.dashboardStats}
// @Router       /dashboard/stats [get]
// @Security     BearerAuth
func (h *DashboardHandler) Stats(c *gin.Context) {
	var stats dashboardStats

	// Count users
	h.db.Model(&model.User{}).Count(&stats.TotalUsers)

	// Count roles
	h.db.Model(&model.Role{}).Count(&stats.TotalRoles)

	// Count files
	h.db.Model(&model.File{}).Count(&stats.TotalFiles)

	// Count total tasks
	h.db.Model(&model.Task{}).Count(&stats.TotalTasks)

	// Count active tasks (pending + running)
	h.db.Model(&model.Task{}).
		Where("status IN ?", []string{"pending", "running"}).
		Count(&stats.ActiveTasks)

	response.OK(c, stats)
}
