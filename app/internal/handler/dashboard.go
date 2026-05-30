package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// DashboardHandler 处理仪表盘统计相关的 HTTP 请求。
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler 创建一个新的 DashboardHandler 实例。
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// Stats 返回系统概览统计数据（用户总数、角色数、文件数和任务数等）。
//
// @Summary      仪表盘统计
// @Description  返回系统概览统计数据
// @Tags         仪表盘
// @Produce      json
// @Success      200  {object}  dto.Response{data=service.DashboardStats}
// @Router       /dashboard/stats [get]
// @Security     BearerAuth
func (h *DashboardHandler) Stats(c *gin.Context) {
	lang, _ := c.Get(middleware.ContextKeyLang)
	langStr, _ := lang.(string)

	stats, err := h.svc.Stats(c.Request.Context(), langStr)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, stats)
}
