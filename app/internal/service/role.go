package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

const permCachePrefix = "perm:"

// RoleService handles business logic for Role operations.
type RoleService struct {
	roleRepo *repository.RoleRepository
	userRepo *repository.UserRepository
	rdb      *redis.Client
}

// NewRoleService creates a new RoleService.
func NewRoleService(roleRepo *repository.RoleRepository, userRepo *repository.UserRepository, rdb *redis.Client) *RoleService {
	return &RoleService{roleRepo: roleRepo, userRepo: userRepo, rdb: rdb}
}

// List returns a paginated list of roles with optional filters.
func (s *RoleService) List(ctx context.Context, req dto.RoleListRequest) ([]model.Role, int64, error) {
	return s.roleRepo.List(ctx, req)
}

// Create creates a new role.
func (s *RoleService) Create(ctx context.Context, req dto.CreateRoleRequest, currentUserID string, isRoot bool) (*model.Role, error) {
	if req.Level < 1 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "")
	}
	// 校验当前用户是否有权限创建指定层级的角色（不能创建高于自身层级的角色）。
	if err := checkRoleLevelChange(ctx, s.userRepo, currentUserID, isRoot, req.Level); err != nil {
		return nil, err
	}

	count, err := s.roleRepo.CountByName(ctx, req.Name, "")
	if err != nil {
		zap.L().Error("check role name uniqueness failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if count > 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "角色名称已存在")
	}

	role := model.Role{
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		Status:      req.Status,
		Level:       req.Level,
	}

	if err := s.roleRepo.Create(ctx, &role); err != nil {
		zap.L().Error("create role failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	return &role, nil
}

// GetByID returns a role by its ID.
func (s *RoleService) GetByID(ctx context.Context, id string) (*model.Role, error) {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "角色不存在")
	}
	return role, nil
}

// Update updates an existing role.
func (s *RoleService) Update(ctx context.Context, id string, req dto.UpdateRoleRequest, currentUserID string, isRoot bool) error {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "角色不存在")
	}

	// 三级权限校验：
	// 1. checkRoleRootGuard — 非 Root 不能修改 Root 级别的角色
	if err := checkRoleRootGuard(role.Level, isRoot); err != nil {
		return err
	}
	// 2. checkRoleHierarchy — 当前用户的角色层级必须 >= 目标角色的层级
	if err := checkRoleHierarchy(ctx, s.userRepo, currentUserID, isRoot, role.Level); err != nil {
		return err
	}
	// 3. checkRoleLevelChange — 如果要修改角色层级，新的层级必须在当前用户的权限范围内
	if req.Level != nil {
		if *req.Level < 1 {
			return apperrors.New(apperrors.ErrBadRequest, "角色层级必须大于0")
		}
		if err := checkRoleLevelChange(ctx, s.userRepo, currentUserID, isRoot, *req.Level); err != nil {
			return err
		}
	}

	// 角色名唯一性校验，排除自身。
	if req.Name != "" && req.Name != role.Name {
		count, err := s.roleRepo.CountByName(ctx, req.Name, id)
		if err != nil {
			zap.L().Error("check role name uniqueness failed", zap.Error(err))
			return apperrors.New(apperrors.ErrInternal, "")
		}
		if count > 0 {
			return apperrors.New(apperrors.ErrBadRequest, "角色名称已存在")
		}
	}

	// 部分更新：仅覆盖请求中提供的字段。
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.SortOrder != nil {
		role.SortOrder = *req.SortOrder
	}
	if req.Status != nil {
		role.Status = *req.Status
	}
	if req.Level != nil {
		role.Level = *req.Level
	}

	if err := s.roleRepo.Update(ctx, role); err != nil {
		zap.L().Error("update role failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}

// Delete deletes a role after checking it's not assigned to users.
func (s *RoleService) Delete(ctx context.Context, id string, currentUserID string, isRoot bool) error {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "角色不存在")
	}

	// 硬性约束：有用户绑定的角色不可删除，防止孤立的角色引用。
	if err := checkRoleRootGuard(role.Level, isRoot); err != nil {
		return err
	}
	if err := checkRoleHierarchy(ctx, s.userRepo, currentUserID, isRoot, role.Level); err != nil {
		return err
	}

	userCount, err := s.roleRepo.CountAssignedUsers(ctx, id)
	if err != nil {
		zap.L().Error("check role usage failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}
	if userCount > 0 {
		return apperrors.New(apperrors.ErrBadRequest, "该角色已分配给用户，无法删除")
	}

	if err := s.roleRepo.Delete(ctx, id); err != nil {
		zap.L().Error("delete role failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	// 删除后清理角色相关的权限缓存，保证下游用户下次请求时重新加载。
	s.invalidatePermCache(ctx)
	return nil
}

// GetPermissions returns the permissions assigned to a role.
func (s *RoleService) GetPermissions(ctx context.Context, id string) ([]model.Permission, error) {
	_, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "角色不存在")
	}

	permissions, err := s.roleRepo.GetPermissionsByRoleID(ctx, id)
	if err != nil {
		zap.L().Error("get role permissions failed", zap.String("role_id", id), zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	return permissions, nil
}

// AssignPermissions replaces all permissions of a role.
func (s *RoleService) AssignPermissions(ctx context.Context, id string, req dto.AssignPermissionsRequest, currentUserID string, isRoot bool) error {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "角色不存在")
	}

	// 必须校验层级：防止低层级用户通过修改高角色权限实现越权。
	if err := checkRoleRootGuard(role.Level, isRoot); err != nil {
		return err
	}
	if err := checkRoleHierarchy(ctx, s.userRepo, currentUserID, isRoot, role.Level); err != nil {
		return err
	}

	// 全量替换角色的权限：先删除旧关联再插入新关联，而非逐条 diff。
	if err := s.roleRepo.ReplacePermissions(ctx, id, req.PermissionIDs); err != nil {
		zap.L().Error("assign permissions failed", zap.String("role_id", id), zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	// 权限变更后立即清除缓存，所有用户下次请求时重新加载。
	s.invalidatePermCache(ctx)
	return nil
}

// invalidatePermCache removes all cached permission entries from Redis.
// 使用 SCAN 游标遍历所有 perm:* 前缀的 key 进行批量删除。
// 相比 FLUSH 或 KEYS，SCAN 不会阻塞 Redis 且支持生产环境大规模 key 的场景。
func (s *RoleService) invalidatePermCache(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var cursor uint64
	var deleted int64
	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, permCachePrefix+"*", 100).Result()
		if err != nil {
			zap.L().Warn("scan perm cache keys failed", zap.Error(err))
			return
		}
		if len(keys) > 0 {
			// 批量删除一次 scan 结果，减少网络往返次数。
			if err := s.rdb.Del(ctx, keys...).Err(); err != nil {
				zap.L().Warn("delete perm cache keys failed", zap.Error(err))
			}
			deleted += int64(len(keys))
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if deleted > 0 {
		zap.L().Info(fmt.Sprintf("invalidated permission cache: %d keys deleted", deleted))
	}
}
