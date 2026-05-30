// Package dto 定义请求和响应的数据传输结构体，包含参数校验和序列化标签。
package dto

// CreatePermissionRequest is the request body for creating a permission.
type CreatePermissionRequest struct {
	Name      string  `json:"name" binding:"required"`
	Code      string  `json:"code" binding:"required"`
	Path      string  `json:"path"`
	Method    string  `json:"method"`
	Type      string  `json:"type" binding:"required"`
	ParentID  *string `json:"parent_id"`
	SortOrder int     `json:"sort_order"`
}

// UpdatePermissionRequest is the request body for updating a permission.
type UpdatePermissionRequest struct {
	Name      string  `json:"name" binding:"omitempty,min=1,max=64"`
	Code      string  `json:"code" binding:"omitempty"`
	Path      string  `json:"path"`
	Method    string  `json:"method"`
	Type      string  `json:"type" binding:"omitempty,oneof=menu button"`
	Icon      string  `json:"icon"`
	ParentID  *string `json:"parent_id"`
	SortOrder *int    `json:"sort_order"`
}
