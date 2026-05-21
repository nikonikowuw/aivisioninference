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

// UserService 处理用户操作的业务逻辑
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService 创建新的 UserService
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// List 返回分页的用户列表，支持可选过滤条件
func (s *UserService) List(ctx context.Context, req dto.UserListRequest) ([]model.User, int64, error) {
	return s.userRepo.List(ctx, req)
}

// Create 创建新用户，包含密码哈希和可选的角色关联
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

	// 在事务中创建用户和角色关联，确保一致性。
	if err := s.userRepo.CreateWithRoles(ctx, &user, req.RoleIDs); err != nil {
		zap.L().Error("create user failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	// 重新查询以返回完整的用户数据（含角色关联），而非直接返回创建后的 user 对象。
	return s.userRepo.FindByIDWithRoles(ctx, user.ID)
}

// GetByID 根据用户ID返回用户
func (s *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "用户不存在")
	}
	return user, nil
}

// Update 更新现有用户
func (s *UserService) Update(ctx context.Context, id string, req dto.UpdateUserRequest, currentUserID string, isRoot bool) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "用户不存在")
	}

	// write=true 表示当前操作需要写入能力，检查层级确保当前用户有足够权限修改目标用户。
	if err := checkUserHierarchy(ctx, s.userRepo, currentUserID, id, isRoot, true); err != nil {
		return err
	}

	// 用户名变更时检查唯一性，排除自身以避免与当前用户名冲突。
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

	// 仅更新请求中非空的字段，实现 PUT 的部分更新语义（类比 PATCH）。
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
	// Status 使用指针类型以便区分「不更新」和「更新为 0」两种状态。
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.userRepo.UpdateWithRoles(ctx, user, req.RoleIDs); err != nil {
		zap.L().Error("update user failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}

// Delete 根据用户ID软删除用户
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

// ResetPassword 允许管理员直接重置指定用户的密码（不需要旧密码），执行层级安全检查。
func (s *UserService) ResetPassword(ctx context.Context, targetUserID, password, currentUserID string, isRoot bool) error {
	// 禁止管理员通过此接口重置自己的密码，防止误操作导致自己无法登录。
	// 自己改密码应走 ChangePassword 流程（需验证旧密码）。
	if targetUserID == currentUserID {
		return apperrors.New(apperrors.ErrBadRequest, "不能重置自己的密码，请使用修改密码功能")
	}

	// 检查目标用户存在
	_, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "用户不存在")
	}

	// 层级权限校验：上级才能重置下级的密码，防止越权操作。
	if err := checkUserHierarchy(ctx, s.userRepo, currentUserID, targetUserID, isRoot, true); err != nil {
		return err
	}

	// 哈希密码
	hashedPassword, err := hash.Hash(password)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	// 更新密码
	if err := s.userRepo.UpdatePassword(ctx, targetUserID, hashedPassword); err != nil {
		zap.L().Error("reset password failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}
