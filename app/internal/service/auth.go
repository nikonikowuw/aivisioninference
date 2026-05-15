package service

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	jwtutil "github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// AuthService handles authentication business logic.
type AuthService struct {
	userRepo   *repository.UserRepository
	rdb        *redis.Client
	jwtManager *jwtutil.Manager
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepository, rdb *redis.Client, jwtManager *jwtutil.Manager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		rdb:        rdb,
		jwtManager: jwtManager,
	}
}

// LoginResult holds the data needed by the handler to complete a login response.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         dto.UserInfo
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*LoginResult, error) {
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, errors.New(errors.ErrUnauthorized, "用户名或密码错误")
	}

	if !hash.Check(req.Password, user.Password) {
		return nil, errors.New(errors.ErrUnauthorized, "用户名或密码错误")
	}

	if user.Status != 1 {
		return nil, errors.New(errors.ErrForbidden, "用户已被禁用")
	}

	var roleIDs []string
	var roleNames []string
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID)
		roleNames = append(roleNames, role.Name)
	}

	accessToken, refreshToken, expiresIn, err := s.jwtManager.GenerateTokenPair(user.ID, roleIDs)
	if err != nil {
		zap.L().Error("generate token pair failed", zap.String("user_id", user.ID), zap.Error(err))
		return nil, errors.New(errors.ErrInternal, "")
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User: dto.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
			Email:       user.Email,
			Status:      user.Status,
			Roles:       roleNames,
			CreatedAt:   user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   user.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// RefreshTokens rotates refresh tokens and returns a new access token.
func (s *AuthService) RefreshTokens(refreshToken string) (accessToken, newRefreshToken string, expiresIn int, err error) {
	return s.jwtManager.RefreshTokens(refreshToken)
}

// RevokeAccessToken revokes a single access token.
func (s *AuthService) RevokeAccessToken(tokenString string) error {
	return s.jwtManager.RevokeAccessToken(tokenString)
}

// RevokeAllRefreshTokens revokes all refresh tokens for a user.
func (s *AuthService) RevokeAllRefreshTokens(userID string) error {
	return s.jwtManager.RevokeAllRefreshTokens(userID)
}

// GetMe returns user info by ID.
func (s *AuthService) GetMe(ctx context.Context, userID string) (*dto.UserInfo, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		return nil, errors.New(errors.ErrNotFound, "用户不存在")
	}

	var roleNames []string
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
	}

	return &dto.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Email:       user.Email,
		Status:      user.Status,
		Roles:       roleNames,
		CreatedAt:   user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ChangePassword verifies old password and updates to new one.
func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New(errors.ErrNotFound, "用户不存在")
	}

	if !hash.Check(oldPassword, user.Password) {
		return errors.New(errors.ErrBadRequest, "旧密码错误")
	}

	hashedPassword, err := hash.Hash(newPassword)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		return errors.New(errors.ErrInternal, "")
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		zap.L().Error("update password failed", zap.Error(err))
		return errors.New(errors.ErrInternal, "")
	}

	if err := s.jwtManager.RevokeAllRefreshTokens(userID); err != nil {
		zap.L().Warn("revoke tokens after password change failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	return nil
}
