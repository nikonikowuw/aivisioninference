// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// permCachePrefix 权限缓存在 Redis 中的键前缀。
const permCachePrefix = "perm:"

// RoleService 处理系统角色相关的业务逻辑
type RoleService struct {
	roleRepo *repository.RoleRepository // 角色数据持久化接口
	userRepo *repository.UserRepository // 用户数据持久化接口
	rdb      *redis.Client              // Redis 客户端，用于管理权限缓存
}

// NewRoleService 创建并返回一个新的 RoleService 实例
func NewRoleService(roleRepo *repository.RoleRepository, userRepo *repository.UserRepository, rdb *redis.Client) *RoleService {
	return &RoleService{roleRepo: roleRepo, userRepo: userRepo, rdb: rdb}
}

// List 根据分页和可选的过滤条件返回角色列表和总条数
func (s *RoleService) List(ctx context.Context, req dto.RoleListRequest) ([]model.Role, int64, error) {
	return s.roleRepo.List(ctx, req)
}

// Create 创建一个新的角色，创建前会验证角色层级合法性及角色名唯一性
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

// GetByID 根据角色 ID 查询角色信息
func (s *RoleService) GetByID(ctx context.Context, id string) (*model.Role, error) {
	role, err := s.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "角色不存在")
	}
	return role, nil
}

// Update 更新现有角色的配置，执行多重越权及层级安全检查，并校验角色名唯一性
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

// Delete 删除指定角色，在删除前校验是否已分配给任何用户，并清除关联的权限缓存
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

// GetPermissions 获取指定角色所拥有的全部权限信息
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

// AssignPermissions 替换指定角色的所有关联权限，更新成功后清除相应的权限缓存
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

// invalidatePermCache 从 Redis 中清除所有带 perm: 前缀的角色及菜单权限缓存。
// 使用 SCAN 游标遍历所有 perm:* 前缀的 Key 进行分批删除，避免使用 KEYS 或 FLUSH 导致 Redis 阻塞。
func (s *RoleService) invalidatePermCache(ctx context.Context) {
	if s.rdb == nil {
		return
	}
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
		zap.L().Info("invalidated permission cache", zap.Int64("deleted", deleted))
	}
}
