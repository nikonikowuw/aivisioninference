package dto

import "github.com/niko-admin/niko-admin/internal/pkg/scopes"

// CreateEdgeNodeTagRequest 创建标签请求
type CreateEdgeNodeTagRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Color       string `json:"color" binding:"max=7"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateEdgeNodeTagRequest 更新标签请求
type UpdateEdgeNodeTagRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Color       string `json:"color" binding:"max=7"`
	Description string `json:"description" binding:"max=500"`
}

// EdgeNodeTagListRequest 标签分页列表请求
type EdgeNodeTagListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
}

// FilterScopes 构建筛选条件
func (r *EdgeNodeTagListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"name", "description"}, r.Keyword))
	}
	return sc
}

// UpdateEdgeNodeTagsRequest 更新节点绑定的标签请求
type UpdateEdgeNodeTagsRequest struct {
	TagIDs []string `json:"tag_ids" binding:"required"`
}
