package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/middleware"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	svc *service.AuthService
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
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

	result, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	c.SetCookie("refresh_token", result.RefreshToken, 7*24*3600, "/", "", false, true)

	response.OK(c, dto.LoginResponse{
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
		User:        result.User,
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
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		response.Err(c, apperrors.New(apperrors.ErrUnauthorized, "缺少刷新令牌"))
		return
	}

	accessToken, newRefreshToken, expiresIn, err := h.svc.RefreshTokens(refreshToken)
	if err != nil {
		zap.L().Warn("refresh token failed", zap.Error(err))
		response.Err(c, apperrors.New(apperrors.ErrTokenInvalid, "刷新令牌无效或已过期"))
		return
	}

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
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && len(authHeader) > 7 {
		tokenString := authHeader[7:]
		if err := h.svc.RevokeAccessToken(tokenString); err != nil {
			zap.L().Warn("revoke access token failed", zap.Error(err))
		}
	}

	refreshToken, _ := c.Cookie("refresh_token")
	if refreshToken != "" {
		userID, exists := c.Get(middleware.ContextKeyUserID)
		if exists {
			if uid, ok := userID.(string); ok {
				if err := h.svc.RevokeAllRefreshTokens(uid); err != nil {
					zap.L().Warn("revoke refresh tokens failed",
						zap.String("user_id", uid),
						zap.Error(err),
					)
				}
			}
		}
	}

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

	info, err := h.svc.GetMe(c.Request.Context(), uid)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, info)
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

	if err := h.svc.ChangePassword(c.Request.Context(), uid, req.OldPassword, req.NewPassword); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}
