package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// EdgeNodeTagService 处理边缘节点标签的业务逻辑
type EdgeNodeTagService struct {
	tagRepo  *repository.EdgeNodeTagRepository
	taskRepo *repository.EdgeScheduledTaskRepository
}

// NewEdgeNodeTagService 创建新的 EdgeNodeTagService
func NewEdgeNodeTagService(tagRepo *repository.EdgeNodeTagRepository, taskRepo *repository.EdgeScheduledTaskRepository) *EdgeNodeTagService {
	return &EdgeNodeTagService{tagRepo: tagRepo, taskRepo: taskRepo}
}

// Create 创建标签
func (s *EdgeNodeTagService) Create(ctx context.Context, req dto.CreateEdgeNodeTagRequest) (*model.EdgeNodeTag, error) {
	item := &model.EdgeNodeTag{
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
	}
	if item.Color == "" {
		item.Color = "#1890ff"
	}
	if err := s.tagRepo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// Update 更新标签
func (s *EdgeNodeTagService) Update(ctx context.Context, id string, req dto.UpdateEdgeNodeTagRequest) error {
	item, err := s.tagRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.New(apperrors.ErrNotFound, "标签不存在")
		}
		return err
	}

	item.Name = req.Name
	item.Color = req.Color
	item.Description = req.Description
	if item.Color == "" {
		item.Color = "#1890ff"
	}
	return s.tagRepo.Update(ctx, item)
}

// Delete 删除标签
func (s *EdgeNodeTagService) Delete(ctx context.Context, id string) error {
	_, err := s.tagRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.New(apperrors.ErrNotFound, "标签不存在")
		}
		return err
	}

	count, err := s.taskRepo.CountByTagID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperrors.Newf(apperrors.ErrBadRequest, "该标签被 %d 个计划任务引用，请先解除引用后再删除", count)
	}

	return s.tagRepo.Delete(ctx, id)
}

// GetByID 查询标签
func (s *EdgeNodeTagService) GetByID(ctx context.Context, id string) (*model.EdgeNodeTag, error) {
	item, err := s.tagRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(apperrors.ErrNotFound, "标签不存在")
		}
		return nil, err
	}
	return item, nil
}

// List 分页查询标签列表
func (s *EdgeNodeTagService) List(ctx context.Context, req dto.EdgeNodeTagListRequest) ([]model.EdgeNodeTag, int64, error) {
	return s.tagRepo.List(ctx, req)
}

// ListAll 返回全部标签
func (s *EdgeNodeTagService) ListAll(ctx context.Context) ([]model.EdgeNodeTag, error) {
	return s.tagRepo.ListAll(ctx)
}

// GetNodesByTagID 查询标签关联的节点 ID 列表
func (s *EdgeNodeTagService) GetNodesByTagID(ctx context.Context, tagID string) ([]string, error) {
	return s.tagRepo.FindNodesByTagID(ctx, tagID)
}

// ReplaceNodeTags 替换节点绑定的标签
func (s *EdgeNodeTagService) ReplaceNodeTags(ctx context.Context, nodeID string, tagIDs []string) error {
	return s.tagRepo.ReplaceNodeTags(ctx, nodeID, tagIDs)
}
