package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/middleware"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/httpx"
	"github.com/niko-admin/niko-admin/internal/pkg/i18n"
	jwtutil "github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	svc         *service.AuthService
	auditLogger middleware.AuditLogger
}

// NewAuthHandler creates a new AuthHandler with the given dependencies.
func NewAuthHandler(svc *service.AuthService, auditLogger middleware.AuditLogger) *AuthHandler {
	return &AuthHandler{svc: svc, auditLogger: auditLogger}
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
// @Failure      401   {object}  dto.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	start := time.Now()

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeLoginAuditLog(c, req.Username, start, http.StatusBadRequest)
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		// 根据错误类型推断 HTTP 状态码
		status := http.StatusUnauthorized
		if appErr, ok := err.(*apperrors.AppError); ok {
			status = middleware.AuditHTTPStatusFromCode(appErr.Code)
		}
		h.writeLoginAuditLog(c, req.Username, start, status)
		attachError(c, err)
		return
	}

	h.writeLoginAuditLog(c, req.Username, start, http.StatusOK)

	c.SetCookie("refresh_token", result.RefreshToken, 7*24*3600, "/", "", httpx.IsSecureRequest(c), true)

	response.OK(c, dto.LoginResponse{
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
		User:        result.User,
	})
}

// writeLoginAuditLog 写入登录审计日志。
func (h *AuthHandler) writeLoginAuditLog(c *gin.Context, username string, start time.Time, httpStatus int) {
	if h.auditLogger == nil {
		return
	}

	resultSummary := "success"
	if httpStatus >= 400 {
		resultSummary = "failed"
	}

	auditLog := &model.AuditLog{
		ResourceType:   "auth",
		RequestPath:    c.FullPath(),
		RequestMethod:  http.MethodPost,
		RequestIP:      c.ClientIP(),
		UserAgent:      middleware.TruncateAuditSummary(c.Request.UserAgent(), 512),
		Username:       username,
		ResultSummary:  resultSummary,
		ResponseStatus: httpStatus,
		DurationMs:     time.Since(start).Milliseconds(),
		ActionType:     i18n.ActionLogin,
	}

	if err := h.auditLogger.Create(c.Request.Context(), auditLog); err != nil {
		zap.L().Warn("write login audit log failed",
			zap.String("username", username),
			zap.Error(err),
		)
	}
}

// Refresh rotates the refresh token and returns a new access token.
//
// @Summary      刷新令牌
// @Description  读取 refresh_token cookie，轮换令牌对
// @Tags         认证管理
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.RefreshResponse}
// @Failure      401  {object}  dto.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		attachError(c, apperrors.New(apperrors.ErrUnauthorized, "缺少刷新令牌"))
		return
	}

	accessToken, newRefreshToken, expiresIn, err := h.svc.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		zap.L().Warn("refresh token failed", zap.Error(err))
		switch {
		case errors.Is(err, jwtutil.ErrRefreshTokenReuse):
			attachError(c, apperrors.New(apperrors.ErrRefreshTokenReuse, "刷新令牌已被复用，所有设备已强制登出"))
		case errors.Is(err, jwtutil.ErrRefreshTokenExpired):
			attachError(c, apperrors.New(apperrors.ErrTokenExpired, "刷新令牌已过期"))
		default:
			attachError(c, apperrors.New(apperrors.ErrTokenInvalid, "刷新令牌无效或已过期"))
		}
		return
	}

	c.SetCookie("refresh_token", newRefreshToken, 7*24*3600, "/", "", httpx.IsSecureRequest(c), true)

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
		if err := h.svc.RevokeAccessToken(c.Request.Context(), tokenString); err != nil {
			zap.L().Warn("revoke access token failed", zap.Error(err))
		}
	}

	refreshToken, _ := c.Cookie("refresh_token")
	if refreshToken != "" {
		userID, exists := c.Get(middleware.ContextKeyUserID)
		if exists {
			if uid, ok := userID.(string); ok {
				if err := h.svc.RevokeAllRefreshTokens(c.Request.Context(), uid); err != nil {
					zap.L().Warn("revoke refresh tokens failed",
						zap.String("user_id", uid),
						zap.Error(err),
					)
				}
			}
		}
	}

	c.SetCookie("refresh_token", "", -1, "/", "", httpx.IsSecureRequest(c), true)
	response.OK(c, nil)
}

// Me returns the current authenticated user's information.
//
// @Summary      获取当前用户信息
// @Description  返回当前登录用户的基本信息和角色列表
// @Tags         认证管理
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.UserInfo}
// @Failure      401  {object}  dto.Response
// @Router       /auth/me [get]
// @Security     BearerAuth
func (h *AuthHandler) Me(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	info, err := h.svc.GetMe(c.Request.Context(), uid)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, info)
}

// UpdateProfile 更新当前登录用户的个人资料
//
// @Summary      更新个人资料
// @Description  允许当前用户更新自己的 display_name、email、avatar_url
// @Tags         认证管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.UpdateProfileRequest  true  "个人资料信息"
// @Success      200   {object}  dto.Response{data=dto.UserInfo}
// @Failure      400   {object}  dto.Response
// @Failure      401   {object}  dto.Response
// @Router       /auth/profile [put]
// @Security     BearerAuth
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	uid, ok := getUserID(c)
	if !ok {
		return
	}

	info, err := h.svc.UpdateProfile(c.Request.Context(), uid, &req)
	if err != nil {
		attachError(c, err)
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
// @Failure      400   {object}  dto.Response
// @Failure      401   {object}  dto.Response
// @Router       /auth/change-password [post]
// @Security     BearerAuth
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
		return
	}

	uid, ok := getUserID(c)
	if !ok {
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), uid, req.OldPassword, req.NewPassword); err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, nil)
}

// UploadAvatar handles avatar file upload for the current user.
//
// @Summary      上传头像
// @Description  当前用户上传头像图片，支持 JPG/PNG/GIF/WebP 格式，最大 2MB
// @Tags         认证管理
// @Accept       multipart/form-data
// @Produce      json
// @Param        avatar  formData  file  true  "头像文件"
// @Success      200     {object}  dto.Response{data=dto.AvatarUploadResponse}
// @Failure      400     {object}  dto.Response
// @Failure      401     {object}  dto.Response
// @Router       /auth/avatar [post]
// @Security     BearerAuth
func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		attachError(c, apperrors.New(apperrors.ErrBadRequest, "缺少头像文件"))
		return
	}

	avatarURL, err := h.svc.UploadAvatar(c.Request.Context(), uid, fileHeader)
	if err != nil {
		attachError(c, err)
		return
	}

	response.OK(c, dto.AvatarUploadResponse{AvatarURL: avatarURL})
}
