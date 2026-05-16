package service

import (
	"context"

	"go.uber.org/zap"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

const noRoleSentinel = 99999

// getUserLevel 获取用户拥有的最小角色层级，即最高权限层级。
// 无角色用户返回最低权限哨兵值；数据库异常会记录日志并返回内部错误。
func getUserLevel(ctx context.Context, userRepo *repository.UserRepository, userID string) (int, error) {
	level, err := userRepo.FindMinRoleLevelByUserID(ctx, userID)
	if err != nil {
		zap.L().Error("get user role level failed", zap.String("user_id", userID), zap.Error(err))
		return 0, apperrors.New(apperrors.ErrInternal, "")
	}
	if level == nil {
		return noRoleSentinel, nil
	}
	return *level, nil
}

// checkRoleHierarchy 校验当前用户是否有权限操作指定层级的角色。
// Root 用户跳过层级校验；权限不足时返回角色层级错误。
func checkRoleHierarchy(ctx context.Context, userRepo *repository.UserRepository, currentUserID string, isRoot bool, targetLevel int) error {
	if isRoot {
		return nil
	}
	currentLevel, err := getUserLevel(ctx, userRepo, currentUserID)
	if err != nil {
		return err
	}
	if currentLevel >= targetLevel {
		return apperrors.New(apperrors.ErrHierarchyLevelRole, "")
	}
	return nil
}

// checkRoleLevelChange 校验当前用户是否有权限设置新的角色层级。
// Root 用户跳过层级校验；新层级高于当前用户权限时返回角色层级错误。
func checkRoleLevelChange(ctx context.Context, userRepo *repository.UserRepository, currentUserID string, isRoot bool, newLevel int) error {
	if isRoot {
		return nil
	}
	currentLevel, err := getUserLevel(ctx, userRepo, currentUserID)
	if err != nil {
		return err
	}
	if newLevel < currentLevel {
		return apperrors.New(apperrors.ErrHierarchyLevelRole, "")
	}
	return nil
}

// checkUserHierarchy 校验当前用户是否有权限操作目标用户。
// Root 用户跳过层级校验；allowSelf 控制是否允许操作自己。
func checkUserHierarchy(ctx context.Context, userRepo *repository.UserRepository, currentUserID, targetUserID string, isRoot bool, allowSelf bool) error {
	if isRoot {
		return nil
	}
	if allowSelf && currentUserID == targetUserID {
		return nil
	}
	currentLevel, err := getUserLevel(ctx, userRepo, currentUserID)
	if err != nil {
		return err
	}
	targetLevel, err := getUserLevel(ctx, userRepo, targetUserID)
	if err != nil {
		return err
	}
	if currentLevel >= targetLevel {
		return apperrors.New(apperrors.ErrHierarchyLevelUser, "")
	}
	return nil
}
