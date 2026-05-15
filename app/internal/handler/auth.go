package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	jwtutil "github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	db  *gorm.DB
	rdb *redis.Client
	jwt *jwtutil.Manager
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
func NewAuthHandler(db *gorm.DB, rdb *redis.Client, jwt *jwtutil.Manager) *AuthHandler {
	return &AuthHandler{db: db, rdb: rdb, jwt: jwt}
}

// Login authenticates a user and returns a token pair.
//
// @Summary      用户登录
// @Description  验证用户名密码，返回 access_token 和 refresh_token cookie
// @Tags         认证管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.LoginRequest  true  "登录信息"
// @Success      200   {object}  dto.Response{data=dto.LoginResponse}
// @Failure      200   {object}  dto.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	// Query user with roles
	var user model.User
	if err := h.db.Preload("Roles").Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, "用户名或密码错误"))
		return
	}

	// Verify password
	if !hash.Check(req.Password, user.Password) {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, "用户名或密码错误"))
		return
	}

	// Check user status
	if user.Status != 1 {
		response.Err(c, apperrors.New(apperrors.ErrForbidden, "用户已被禁用"))
		return
	}

	// Collect role IDs and names
	var roleIDs []string
	var roleNames []string
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID)
		roleNames = append(roleNames, role.Name)
	}

	// Generate token pair
	accessToken, refreshToken, expiresIn, err := h.jwt.GenerateTokenPair(user.ID, roleIDs)
	if err != nil {
		zap.L().Error("generate token pair failed",
			zap.String("user_id", user.ID),
			zap.Error(err),
		)
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Set refresh token cookie (7 days, HttpOnly, Secure in production)
	c.SetCookie("refresh_token", refreshToken, 7*24*3600, "/", "", false, true)

	response.OK(c, dto.LoginResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
		User: dto.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Roles:       roleNames,
		},
	})
}

// Refresh rotates the refresh token and returns a new access token.
//
// @Summary      刷新令牌
// @Description  读取 refresh_token cookie，轮换令牌对
// @Tags         认证管理
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.RefreshResponse}
// @Failure      200  {object}  dto.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	// Read refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, "缺少刷新令牌"))
		return
	}

	// Rotate tokens
	accessToken, newRefreshToken, expiresIn, err := h.jwt.RefreshTokens(refreshToken)
	if err != nil {
		zap.L().Warn("refresh token failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrTokenInvalid, "刷新令牌无效或已过期"))
		return
	}

	// Set new refresh token cookie
	c.SetCookie("refresh_token", newRefreshToken, 7*24*3600, "/", "", false, true)

	response.OK(c, dto.RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
	})
}

// Logout revokes the current token and clears the refresh token cookie.
//
// @Summary      退出登录
// @Description  撤销当前令牌，清除 refresh_token cookie
// @Tags         认证管理
// @Produce      json
// @Success      200  {object}  dto.Response
// @Router       /auth/logout [post]
// @Security     BearerAuth
func (h *AuthHandler) Logout(c *gin.Context) {
	// Revoke access token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && len(authHeader) > 7 {
		tokenString := authHeader[7:]
		if err := h.jwt.RevokeAccessToken(tokenString); err != nil {
			zap.L().Warn("revoke access token failed", zap.Error(err))
		}
	}

	// Revoke refresh token from cookie
	refreshToken, _ := c.Cookie("refresh_token")
	if refreshToken != "" {
		// Revoke all refresh tokens for the current user
		userID, exists := c.Get(middleware.ContextKeyUserID)
		if exists {
			if uid, ok := userID.(string); ok {
				if err := h.jwt.RevokeAllRefreshTokens(uid); err != nil {
					zap.L().Warn("revoke refresh tokens failed",
						zap.String("user_id", uid),
						zap.Error(err),
					)
				}
			}
		}
	}

	// Clear refresh token cookie
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.OK(c, nil)
}

// Me returns the current authenticated user's information.
//
// @Summary      获取当前用户信息
// @Description  返回当前登录用户的基本信息和角色列表
// @Tags         认证管理
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.UserInfo}
// @Failure      200  {object}  dto.Response
// @Router       /auth/me [get]
// @Security     BearerAuth
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, ""))
		return
	}

	uid, ok := userID.(string)
	if !ok || uid == "" {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, ""))
		return
	}

	var user model.User
	if err := h.db.Preload("Roles").Where("id = ?", uid).First(&user).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "用户不存在"))
		return
	}

	var roleNames []string
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
	}

	response.OK(c, dto.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Roles:       roleNames,
	})
}

// ChangePassword validates the old password and updates to a new one.
//
// @Summary      修改密码
// @Description  验证旧密码后更新为新密码
// @Tags         认证管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.ChangePasswordRequest  true  "密码信息"
// @Success      200   {object}  dto.Response
// @Failure      200   {object}  dto.Response
// @Router       /auth/change-password [post]
// @Security     BearerAuth
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	userID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, ""))
		return
	}

	uid, ok := userID.(string)
	if !ok || uid == "" {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, ""))
		return
	}

	// Fetch current user
	var user model.User
	if err := h.db.Where("id = ?", uid).First(&user).Error; err != nil {
		response.Err(c, apperrors.New(apperrors.ErrNotFound, "用户不存在"))
		return
	}

	// Verify old password
	if !hash.Check(req.OldPassword, user.Password) {
		response.Err(c, apperrors.New(apperrors.ErrBadRequest, "旧密码错误"))
		return
	}

	// Hash new password
	hashedPassword, err := hash.Hash(req.NewPassword)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Update password
	if err := h.db.Model(&user).Update("password", hashedPassword).Error; err != nil {
		zap.L().Error("update password failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
		return
	}

	// Revoke all existing tokens for security
	if err := h.jwt.RevokeAllRefreshTokens(uid); err != nil {
		zap.L().Warn("revoke tokens after password change failed",
			zap.String("user_id", uid),
			zap.Error(err),
		)
	}

	response.OK(c, nil)
}
