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
