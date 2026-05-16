package dto

import "github.com/niko-admin/niko-admin/internal/pkg/scopes"

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Username    string   `json:"username" binding:"required,min=2,max=32"`
	Password    string   `json:"password" binding:"required,min=6,max=72"`
	Email       string   `json:"email" binding:"omitempty,email"`
	DisplayName string   `json:"display_name" binding:"max=64"`
	AvatarURL   string   `json:"avatar_url" binding:"max=512"`
	Status      int      `json:"status"`
	RoleIDs     []string `json:"role_ids"`
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Username    string   `json:"username" binding:"omitempty,min=2,max=32"`
	Email       string   `json:"email" binding:"omitempty,email"`
	DisplayName string   `json:"display_name" binding:"omitempty,max=64"`
	AvatarURL   string   `json:"avatar_url" binding:"omitempty,max=512"`
	Status      *int     `json:"status"`
	RoleIDs     []string `json:"role_ids"`
}

// UserListRequest is the request for listing users with filters.
type UserListRequest struct {
	PageRequest
	Keyword string `form:"keyword"`
	Status  *int   `form:"status"`
}

func (r *UserListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"username", "display_name", "email"}, r.Keyword))
	}
	if r.Status != nil {
		sc = append(sc, scopes.Eq("status", *r.Status))
	}
	return sc
}
