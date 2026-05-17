package dto

import "github.com/niko-admin/niko-admin/internal/pkg/scopes"

// CreateRoleRequest is the request body for creating a role.
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	Status      int    `json:"status"`
	Level       int    `json:"level" binding:"required,gte=1"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=64"`
	Description string `json:"description" binding:"max=256"`
	SortOrder   *int   `json:"sort_order"`
	Status      *int   `json:"status"`
	Level       *int   `json:"level" binding:"omitempty,gte=1"`
}

// AssignPermissionsRequest is the request body for assigning permissions to a role.
type AssignPermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids" binding:"required"`
}

// RoleListRequest is the request for listing roles with filters.
type RoleListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
	Status  *int   `form:"status"`
}

func (r *RoleListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"name", "description"}, r.Keyword))
	}
	if r.Status != nil {
		sc = append(sc, scopes.Eq("status", *r.Status))
	}
	return sc
}
