package dto

import "github.com/niko-admin/niko-admin/internal/pkg/scopes"

// CreateRoleRequest is the request body for creating a role.
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	Status      int    `json:"status"`
}

// UpdateRoleRequest is the request body for updating a role.
type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=64"`
	Description string `json:"description" binding:"max=256"`
	SortOrder   *int   `json:"sort_order"`
	Status      *int   `json:"status"`
}

// AssignPermissionsRequest is the request body for assigning permissions to a role.
type AssignPermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids" binding:"required"`
}

// RoleListRequest is the request for listing roles with filters.
type RoleListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
}

func (r *RoleListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"name", "description"}, r.Keyword))
	}
	return sc
}
