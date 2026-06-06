// Package dto 定义请求和响应的数据传输结构体，包含参数校验和序列化标签。
package dto

// LoginRequest is the login request body.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the successful login response.
type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	ExpiresIn   int      `json:"expires_in"`
	User        UserInfo `json:"user"`
}

// RoleInfo 角色信息
type RoleInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Level       int    `json:"level"`
}

// Menu 用户菜单项
type Menu struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Path      string `json:"path"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sort_order"`
	Children  []Menu `json:"children,omitempty"`
}

// UserInfo contains basic user information returned after auth.
type UserInfo struct {
	ID              string     `json:"id"`
	Username        string     `json:"username"`
	DisplayName     string     `json:"display_name"`
	AvatarURL       string     `json:"avatar_url"`
	Email           string     `json:"email"`
	EmailVerified   bool       `json:"email_verified"`
	Status          int        `json:"status"`
	Roles           []RoleInfo `json:"roles"`
	Menus           []Menu     `json:"menus"`
	PermissionCodes []string   `json:"permission_codes"`
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
}

// RefreshResponse is the token refresh response.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// ChangePasswordRequest is the password change request body.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=72"`
}

// RequestPasswordResetRequest requests a password reset email.
type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ConfirmPasswordResetRequest resets password with a one-time token.
type ConfirmPasswordResetRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=72"`
}

// UpdateProfileRequest 用户自助更新个人资料请求
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name"`
	Email       *string `json:"email" binding:"omitempty,email"`
	AvatarURL   *string `json:"avatar_url" binding:"omitempty,url"`
}
