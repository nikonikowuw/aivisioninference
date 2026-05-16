package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// UserService handles business logic for User operations.
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// List returns a paginated list of users with optional filters.
func (s *UserService) List(ctx context.Context, req dto.UserListRequest) ([]model.User, int64, error) {
	return s.userRepo.List(ctx, req)
}

// Create creates a new user with password hashing and optional role association.
func (s *UserService) Create(ctx context.Context, req dto.CreateUserRequest) (*model.User, error) {
	count, err := s.userRepo.CountByUsername(ctx, req.Username, "")
	if err != nil {
		zap.L().Error("check username uniqueness failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if count > 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "用户名已存在")
	}

	hashedPassword, err := hash.Hash(req.Password)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	user := model.User{
		Username:    req.Username,
		Password:    hashedPassword,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
		Status:      req.Status,
	}

	if err := s.userRepo.CreateWithRoles(ctx, &user, req.RoleIDs); err != nil {
		zap.L().Error("create user failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	return s.userRepo.FindByIDWithRoles(ctx, user.ID)
}

// GetByID returns a user by its ID.
func (s *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "用户不存在")
	}
	return user, nil
}

// Update updates an existing user.
func (s *UserService) Update(ctx context.Context, id string, req dto.UpdateUserRequest, currentUserID string, isRoot bool) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "用户不存在")
	}

	if err := checkUserHierarchy(ctx, s.userRepo, currentUserID, id, isRoot, true); err != nil {
		return err
	}

	if req.Username != "" && req.Username != user.Username {
		count, err := s.userRepo.CountByUsername(ctx, req.Username, id)
		if err != nil {
			zap.L().Error("check username uniqueness failed", zap.Error(err))
			return apperrors.New(apperrors.ErrInternal, "")
		}
		if count > 0 {
			return apperrors.New(apperrors.ErrBadRequest, "用户名已存在")
		}
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.userRepo.UpdateWithRoles(ctx, user, req.RoleIDs); err != nil {
		zap.L().Error("update user failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}

// Delete soft-deletes a user by its ID.
func (s *UserService) Delete(ctx context.Context, id, currentUserID string, isRoot bool) error {
	if id == currentUserID {
		return apperrors.New(apperrors.ErrBadRequest, "不能删除当前登录用户")
	}

	if err := checkUserHierarchy(ctx, s.userRepo, currentUserID, id, isRoot, false); err != nil {
		return err
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		zap.L().Error("delete user failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}
	return nil
}
