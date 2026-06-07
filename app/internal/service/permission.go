// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	cachepkg "github.com/niko-admin/niko-admin/internal/pkg/cache"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// permissionTreeCacheKey 权限树数据在缓存中的键名。
const permissionTreeCacheKey = "perm:tree:all"

// PermissionService 处理系统权限相关的业务逻辑
type PermissionService struct {
	permRepo *repository.PermissionRepository // 权限数据持久化接口
	cache    cachepkg.Cache                   // 缓存管理器实例
}

// NewPermissionService 创建并返回一个新的 PermissionService 实例
func NewPermissionService(permRepo *repository.PermissionRepository, cache cachepkg.Cache) *PermissionService {
	return &PermissionService{permRepo: permRepo, cache: cache}
}

// Tree 获取并返回所有权限的树形结构数据（带 Redis 缓存支持）
func (s *PermissionService) Tree(ctx context.Context) ([]model.Permission, error) {
	if s.cache != nil {
		cached, err := s.cache.Get(ctx, permissionTreeCacheKey)
		if err == nil {
			var tree []model.Permission
			unmarshalErr := json.Unmarshal(cached, &tree)
			if unmarshalErr == nil {
				return tree, nil
			}
			zap.L().Warn("unmarshal permission tree cache failed", zap.Error(unmarshalErr))
		}
	}

	all, err := s.permRepo.FindAllOrdered(ctx)
	if err != nil {
		zap.L().Error("list permissions failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	tree := buildPermissionTree(all, nil)

	if s.cache != nil {
		if data, marshalErr := json.Marshal(tree); marshalErr == nil {
			if setErr := s.cache.Set(ctx, permissionTreeCacheKey, data, 10*time.Minute); setErr != nil {
				zap.L().Warn("set permission tree cache failed", zap.Error(setErr))
			}
		} else {
			zap.L().Warn("marshal permission tree cache failed", zap.Error(marshalErr))
		}
	}

	return tree, nil
}

// buildPermissionTree 递归地将扁平的权限列表构建为树形嵌套结构
func buildPermissionTree(all []model.Permission, parentID *string) []model.Permission {
	var result []model.Permission
	for _, p := range all {
		if (parentID == nil && p.ParentID == nil) ||
			(parentID != nil && p.ParentID != nil && *parentID == *p.ParentID) {
			node := p
			node.Children = buildPermissionTree(all, &node.ID)
			result = append(result, node)
		}
	}
	return result
}

// invalidateTreeCache 清除 Redis 中缓存的权限树数据
func (s *PermissionService) invalidateTreeCache(ctx context.Context) {
	if s.cache != nil {
		if err := s.cache.Del(ctx, permissionTreeCacheKey); err != nil {
			zap.L().Warn("invalidate permission tree cache failed", zap.Error(err))
		}
	}
}

// applyOptionalPermissionUpdates applies optional parent ID and sort order updates.
func applyOptionalPermissionUpdates(perm *model.Permission, parentID *string, sortOrder *int) {
	if parentID != nil {
		perm.ParentID = parentID
	}
	if sortOrder != nil {
		perm.SortOrder = *sortOrder
	}
}

// Update 更新现有的权限配置，更新成功后清除权限树缓存
func (s *PermissionService) Update(ctx context.Context, id string, req dto.UpdatePermissionRequest) error {
	perm, err := s.permRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrPermissionNotFound, "")
	}

	if req.Code != "" && req.Code != perm.Code {
		count, err := s.permRepo.CountByCode(ctx, req.Code)
		if err != nil {
			zap.L().Error("check permission code uniqueness failed", zap.Error(err))
			return apperrors.New(apperrors.ErrInternal, "")
		}
		if count > 0 {
			return apperrors.New(apperrors.ErrPermissionCodeTaken, "")
		}
	}

	if req.Name != "" {
		perm.Name = req.Name
	}
	if req.Code != "" {
		perm.Code = req.Code
	}
	if req.Path != "" {
		perm.Path = req.Path
	}
	if req.Method != "" {
		perm.Method = req.Method
	}
	if req.Type != "" {
		perm.Type = req.Type
	}
	if req.Icon != "" {
		perm.Icon = req.Icon
	}
	applyOptionalPermissionUpdates(perm, req.ParentID, req.SortOrder)

	if err := s.permRepo.Update(ctx, perm); err != nil {
		zap.L().Error("update permission failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	s.invalidateTreeCache(ctx)
	return nil
}

// Delete 删除指定权限及其所有子孙权限，执行前会校验其是否已分配给任何角色
func (s *PermissionService) Delete(ctx context.Context, id string) error {
	if _, err := s.permRepo.FindByID(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrPermissionNotFound, "")
	}

	descendants, err := s.permRepo.FindDescendantIDs(ctx, id)
	if err != nil {
		zap.L().Error("find descendant permissions failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	checkIDs := append([]string{id}, descendants...)
	for _, pid := range checkIDs {
		roleCount, err := s.permRepo.CountAssignedRoles(ctx, pid)
		if err != nil {
			zap.L().Error("check permission usage failed", zap.Error(err))
			return apperrors.New(apperrors.ErrInternal, "")
		}
		if roleCount > 0 {
			return apperrors.New(apperrors.ErrPermissionAssigned, "")
		}
	}

	if err := s.permRepo.Delete(ctx, id); err != nil {
		zap.L().Error("delete permission failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	s.invalidateTreeCache(ctx)
	return nil
}

// Create 在校验父节点存在性与编码唯一性后，创建新的权限记录
func (s *PermissionService) Create(ctx context.Context, req dto.CreatePermissionRequest) (*model.Permission, error) {
	count, err := s.permRepo.CountByCode(ctx, req.Code)
	if err != nil {
		zap.L().Error("check permission code uniqueness failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if count > 0 {
		return nil, apperrors.New(apperrors.ErrPermissionCodeTaken, "")
	}

	if req.ParentID != nil {
		exists, err := s.permRepo.ExistsByID(ctx, *req.ParentID)
		if err != nil {
			zap.L().Error("check parent permission failed", zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		if !exists {
			return nil, apperrors.New(apperrors.ErrPermissionNotFound, "")
		}
	}

	item := model.Permission{
		Name:      req.Name,
		Code:      req.Code,
		Path:      req.Path,
		Method:    req.Method,
		Type:      req.Type,
		ParentID:  req.ParentID,
		SortOrder: req.SortOrder,
	}

	if err := s.permRepo.Create(ctx, &item); err != nil {
		zap.L().Error("create permission failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	s.invalidateTreeCache(ctx)
	return &item, nil
}
