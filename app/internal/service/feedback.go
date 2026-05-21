package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	mailpkg "github.com/niko-admin/niko-admin/internal/pkg/mail"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// FeedbackService 处理用户反馈相关的业务逻辑
type FeedbackService struct {
	repo    feedbackServiceRepo // 反馈数据访问接口
	mailSvc *MailService        // 邮件服务实例，用于发送通知
}

// feedbackServiceRepo 定义反馈服务依赖的数据持久化接口（方便进行 mock 测试）
type feedbackServiceRepo interface {
	Create(ctx context.Context, item *model.Feedback) error
	FindByID(ctx context.Context, id string) (*model.Feedback, error)
	Update(ctx context.Context, item *model.Feedback) error
	List(ctx context.Context, req dto.FeedbackListRequest) ([]model.Feedback, int64, error)
}

// NewFeedbackService 创建并返回一个新的 FeedbackService 实例
func NewFeedbackService(repo *repository.FeedbackRepository, mailSvc *MailService) *FeedbackService {
	return &FeedbackService{repo: repo, mailSvc: mailSvc}
}

// Create 创建一条新的反馈记录，并尝试通过邮件通知管理员
func (s *FeedbackService) Create(ctx context.Context, userID string, req dto.FeedbackCreateRequest) (*model.Feedback, error) {
	item := &model.Feedback{
		Source:   model.FeedbackSourceUser,
		Category: req.Category,
		Title:    req.Title,
		Content:  req.Content,
		UserID:   &userID,
		Status:   model.FeedbackStatusOpen,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		zap.L().Error("create feedback failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	if s.mailSvc != nil {
		to, err := s.mailSvc.NotificationAddress(ctx)
		if err == nil && to != "" {
			if err := s.mailSvc.Send(ctx, mailpkg.Message{
				To:       []string{to},
				Subject:  "New user feedback: " + req.Title,
				TextBody: req.Content,
			}); err != nil {
				zap.L().Warn("send feedback notification failed", zap.String("feedback_id", item.ID), zap.Error(sanitizeMailError(err)))
			}
		}
	}
	return item, nil
}

// List 根据分页和可选的过滤条件返回反馈记录列表和总数
func (s *FeedbackService) List(ctx context.Context, req dto.FeedbackListRequest) ([]model.Feedback, int64, error) {
	if req.StartTime != "" || req.EndTime != "" {
		from, to, err := scopes.ParseTimeRange(req.StartTime, req.EndTime)
		if err != nil {
			return nil, 0, mapTimeRangeError(err)
		}
		req.FromTime = from
		req.ToTime = to
	}
	return s.repo.List(ctx, req)
}

// UpdateStatus 更新反馈的流转状态，并记录处理人和处理时间
func (s *FeedbackService) UpdateStatus(ctx context.Context, id, status, handlerID string) (*model.Feedback, error) {
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrFeedbackNotFound, "")
	}
	item.Status = status
	item.HandledBy = &handlerID
	now := time.Now()
	item.HandledAt = &now
	if err := s.repo.Update(ctx, item); err != nil {
		zap.L().Error("update feedback status failed", zap.String("feedback_id", id), zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return item, nil
}
