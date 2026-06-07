// Package handler 提供 HTTP 请求处理层（Controller），负责参数绑定、校验和响应返回。
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

// AuthHandler 处理用户认证相关的 HTTP 请求。
type AuthHandler struct {
	svc         *service.AuthService
	emailSvc    *service.EmailVerificationService
	auditLogger middleware.AuditLogger
}

// NewAuthHandler 创建一个新的 AuthHandler 实例。
func NewAuthHandler(svc *service.AuthService, auditLogger middleware.AuditLogger) *AuthHandler {
	return &AuthHandler{svc: svc, auditLogger: auditLogger}
}

// NewAuthHandlerWithEmail 创建支持邮件服务认证的 AuthHandler 实例。
func NewAuthHandlerWithEmail(svc *service.AuthService, emailSvc *service.EmailVerificationService, auditLogger middleware.AuditLogger) *AuthHandler {
	return &AuthHandler{svc: svc, emailSvc: emailSvc, auditLogger: auditLogger}
}

// Login 验证用户名和密码，并在登录成功后返回访问令牌和刷新令牌。
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
	// 绑定并验证登录请求的 JSON 参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.writeLoginAuditLog(c, req.Username, start, http.StatusBadRequest)
		attachError(c, badRequestError(c, err))
		return
	}

	// 调用服务层进行登录验证
	result, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		// 根据错误类型推断 HTTP 状态码
		status := http.StatusUnauthorized
		if appErr, ok := err.(*apperrors.AppError); ok {
			status = middleware.AuditHTTPStatusFromCode(appErr.Code)
		}
		// 登录失败，写入失败审计日志
		h.writeLoginAuditLog(c, req.Username, start, status)
		attachError(c, err)
		return
	}

	// 登录成功，写入成功审计日志
	h.writeLoginAuditLog(c, req.Username, start, http.StatusOK)

	// 将 refresh_token 设置到 HttpOnly Cookie 中，提供高安全性保护
	c.SetCookie("refresh_token", result.RefreshToken, 7*24*3600, "/", "", httpx.IsSecureRequest(c), true)

	// 返回登录成功的 Access Token 及其过期时间等信息
	response.OK(c, dto.LoginResponse{
		AccessToken: result.AccessToken,
		ExpiresIn:   result.ExpiresIn,
		User:        result.User,
	})
}

// writeLoginAuditLog 手动写入登录审计日志。
// 登录发生在 JWT 认证之前，中间件的 Audit 无法捕获用户身份，因此由 Handler 主动记录。
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

// Refresh 使用客户端持有的 refresh_token，安全刷新并轮换生成新的访问令牌对。
//
// @Summary      刷新令牌
// @Description  读取 refresh_token cookie，轮换令牌对
// @Tags         认证管理
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.RefreshResponse}
// @Failure      401  {object}  dto.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	// 从安全 Cookie 中读取当前的 refresh_token
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		attachError(c, apperrors.New(apperrors.ErrUnauthorized, "缺少刷新令牌"))
		return
	}

	// 在服务中轮换并刷新令牌，防止并发复用攻击
	accessToken, newRefreshToken, expiresIn, err := h.svc.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		zap.L().Warn("refresh token failed", zap.Error(err))
		// 区分三种令牌异常原因，返回不同的用户提示，帮助定位问题。
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

	// HttpOnly + Secure 的 refresh_token cookie，前端 JS 不可读写，防止 XSS 窃取。
	c.SetCookie("refresh_token", newRefreshToken, 7*24*3600, "/", "", httpx.IsSecureRequest(c), true)

	response.OK(c, dto.RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
	})
}

// Logout 销毁或注销当前会话的 access_token，并清空 refresh_token cookie。
//
// @Summary      退出登录
// @Description  撤销当前令牌，清除 refresh_token cookie
// @Tags         认证管理
// @Produce      json
// @Success      200  {object}  dto.Response
// @Router       /auth/logout [post]
// @Security     BearerAuth
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从 Authorization header 提取 Bearer token，去掉 "Bearer " 前缀（固定 7 字符）。
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && len(authHeader) > 7 {
		tokenString := authHeader[7:]
		// 吊销当前请求对应的 access_token（加入黑名单）
		if err := h.svc.RevokeAccessToken(c.Request.Context(), tokenString); err != nil {
			zap.L().Warn("revoke access token failed", zap.Error(err))
		}
	}

	// 吊销该用户的所有 refresh token，实现「在所有设备上登出」的安全语义。
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

	// 清除客户端的 refresh_token cookie（MaxAge=-1 表示立即删除）。
	c.SetCookie("refresh_token", "", -1, "/", "", httpx.IsSecureRequest(c), true)
	response.OK(c, nil)
}

// Me 返回当前已认证用户的基本资料与角色权限。
//
// @Summary      获取当前用户信息
// @Description  返回当前登录用户的基本信息 and 角色列表
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

// UpdateProfile 更新当前登录用户的显示名称、邮箱和头像 URL 等基本属性。
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
		attachError(c, badRequestError(c, err))
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

// ChangePassword 验证旧密码无误后，安全地更改用户的登录密码。
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
		attachError(c, badRequestError(c, err))
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

// RequestPasswordReset 发送找回密码/重置密码的邮件至用户邮箱。
//
// @Summary      请求找回密码
// @Description  发送密码重置邮件；响应不泄露邮箱是否已注册
// @Tags         认证管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.RequestPasswordResetRequest  true  "邮箱"
// @Success      200   {object}  dto.Response
// @Router       /auth/password-reset/request [post]
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req dto.RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	if h.emailSvc != nil {
		if err := h.emailSvc.SendPasswordReset(c.Request.Context(), req.Email, c.ClientIP()); err != nil {
			attachError(c, err)
			return
		}
	}
	response.OK(c, nil)
}

// ResetPassword 校验一次性令牌（Token）并设置新的用户登录密码。
//
// @Summary      重置密码
// @Description  使用邮件令牌设置新密码
// @Tags         认证管理
// @Accept       json
// @Produce      json
// @Param        body  body  dto.ConfirmPasswordResetRequest  true  "重置信息"
// @Success      200   {object}  dto.Response
// @Router       /auth/password-reset/confirm [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ConfirmPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	if h.emailSvc == nil {
		attachError(c, apperrors.New(apperrors.ErrMailNotEnabled, ""))
		return
	}
	if err := h.emailSvc.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// UploadAvatar 处理当前用户的头像上传请求，校验大小格式，并持久化头像文件。
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
