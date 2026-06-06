// AITimeSchedule DTO 定义
package dto

import (
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// CreateAITimeScheduleRequest 创建时间配置请求
type CreateAITimeScheduleRequest struct {
	Name        string       `json:"name" binding:"required,max=255"`
	Description string       `json:"description" binding:"max=500"`
	StartDate   string       `json:"start_date" binding:"required"`
	EndDate     string       `json:"end_date" binding:"required"`
	TimeWindows []TimeWindow `json:"time_windows" binding:"required,min=1"`
}

// UpdateAITimeScheduleRequest 更新时间配置请求
type UpdateAITimeScheduleRequest struct {
	Name        string       `json:"name" binding:"required,max=255"`
	Description string       `json:"description" binding:"max=500"`
	StartDate   string       `json:"start_date" binding:"required"`
	EndDate     string       `json:"end_date" binding:"required"`
	TimeWindows []TimeWindow `json:"time_windows" binding:"required,min=1"`
}

// AITimeScheduleListRequest 时间配置分页列表请求
type AITimeScheduleListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
}

// FilterScopes 构建筛选条件
func (r *AITimeScheduleListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"name", "description"}, r.Keyword))
	}
	return sc
}
