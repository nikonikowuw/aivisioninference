package dto

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Username    string   `json:"username" binding:"required"`
	Password    string   `json:"password" binding:"required"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	AvatarURL   string   `json:"avatar_url"`
	Status      int      `json:"status"`
	RoleIDs     []string `json:"role_ids"`
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	AvatarURL   string   `json:"avatar_url"`
	Status      int      `json:"status"`
	RoleIDs     []string `json:"role_ids"`
}
