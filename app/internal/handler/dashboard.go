package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// DashboardHandler handles dashboard statistics HTTP requests.
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler with the given dependencies.
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// Stats returns dashboard statistics (total users, roles, files, tasks).
//
// @Summary      仪表盘统计
// @Description  返回系统概览统计数据
// @Tags         仪表盘
// @Produce      json
// @Success      200  {object}  dto.Response{data=service.DashboardStats}
// @Router       /dashboard/stats [get]
// @Security     BearerAuth
func (h *DashboardHandler) Stats(c *gin.Context) {
	stats, err := h.svc.Stats(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, stats)
}
