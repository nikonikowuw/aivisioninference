package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// PermissionService handles business logic for Permission operations.
type PermissionService struct {
	permRepo *repository.PermissionRepository
}

// NewPermissionService creates a new PermissionService.
func NewPermissionService(permRepo *repository.PermissionRepository) *PermissionService {
	return &PermissionService{permRepo: permRepo}
}

// Tree returns all permissions as a tree structure.
func (s *PermissionService) Tree(ctx context.Context) ([]model.Permission, error) {
	all, err := s.permRepo.FindAllOrdered(ctx)
	if err != nil {
		zap.L().Error("list permissions failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return buildPermissionTree(all, nil), nil
}

// buildPermissionTree recursively builds a permission tree from a flat list.
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

// Create creates a new permission after validation.
func (s *PermissionService) Create(ctx context.Context, req dto.CreatePermissionRequest) (*model.Permission, error) {
	count, err := s.permRepo.CountByCode(ctx, req.Code)
	if err != nil {
		zap.L().Error("check permission code uniqueness failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if count > 0 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "权限编码已存在")
	}

	if req.ParentID != nil {
		exists, err := s.permRepo.ExistsByID(ctx, *req.ParentID)
		if err != nil {
			zap.L().Error("check parent permission failed", zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		if !exists {
			return nil, apperrors.New(apperrors.ErrNotFound, "父级权限不存在")
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

	return &item, nil
}
