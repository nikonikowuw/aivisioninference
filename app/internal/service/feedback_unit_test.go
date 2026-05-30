package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

type fakeFeedbackServiceRepo struct {
	items map[string]*model.Feedback
}

func (f *fakeFeedbackServiceRepo) Create(_ context.Context, item *model.Feedback) error {
	if f.items == nil {
		f.items = map[string]*model.Feedback{}
	}
	if item.ID == "" {
		item.ID = time.Now().Format("20060102150405.000000000")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	f.items[item.ID] = item
	return nil
}

func (f *fakeFeedbackServiceRepo) FindByID(_ context.Context, id string) (*model.Feedback, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return item, nil
}

func (f *fakeFeedbackServiceRepo) Update(_ context.Context, item *model.Feedback) error {
	f.items[item.ID] = item
	return nil
}

func (f *fakeFeedbackServiceRepo) List(_ context.Context, req dto.FeedbackListRequest) ([]model.Feedback, int64, error) {
	list := f.filter(req)
	return list, int64(len(list)), nil
}

func (f *fakeFeedbackServiceRepo) ListForExport(_ context.Context, req dto.FeedbackListRequest, _ int) ([]model.Feedback, error) {
	return f.filter(req), nil
}

func (f *fakeFeedbackServiceRepo) filter(req dto.FeedbackListRequest) []model.Feedback {
	list := make([]model.Feedback, 0, len(f.items))
	for _, it := range f.items {
		if req.Source != "" && it.Source != req.Source {
			continue
		}
		if req.Status != "" && it.Status != req.Status {
			continue
		}
		if req.Keyword != "" && it.Title != req.Keyword && it.Category != req.Keyword && it.Email != req.Keyword {
			continue
		}
		if req.FromTime != nil && it.CreatedAt.Before(*req.FromTime) {
			continue
		}
		if req.ToTime != nil && it.CreatedAt.After(*req.ToTime) {
			continue
		}
		list = append(list, *it)
	}
	return list
}

func TestFeedbackServiceCreate(t *testing.T) {
	repo := &fakeFeedbackServiceRepo{}
	svc := &FeedbackService{repo: repo}

	item, err := svc.Create(context.Background(), "u-1", dto.FeedbackCreateRequest{
		Category: "bug",
		Title:    "cannot submit",
		Content:  "steps...",
	})
	require.NoError(t, err)
	require.Equal(t, model.FeedbackSourceUser, item.Source)
	require.Equal(t, model.FeedbackStatusOpen, item.Status)
	require.NotNil(t, item.UserID)
	require.Equal(t, "u-1", *item.UserID)
}

func TestFeedbackServiceListFilters(t *testing.T) {
	repo := &fakeFeedbackServiceRepo{items: map[string]*model.Feedback{
		"1": {
			BaseModel: model.BaseModel{ID: "1", CreatedAt: time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)},
			Source:    model.FeedbackSourceUser,
			Category:  "bug",
			Title:     "login issue",
			Email:     "a@example.com",
			Status:    model.FeedbackStatusOpen,
		},
		"2": {
			BaseModel: model.BaseModel{ID: "2", CreatedAt: time.Date(2026, 1, 2, 13, 0, 0, 0, time.UTC)},
			Source:    model.FeedbackSourceEmail,
			Category:  "payment",
			Title:     "payment question",
			Email:     "b@example.com",
			Status:    model.FeedbackStatusResolved,
		},
	}}
	svc := &FeedbackService{repo: repo}

	list, total, err := svc.List(context.Background(), dto.FeedbackListRequest{
		PageRequest: dto.PageRequest{Page: 1, PageSize: 20},
		Keyword:     "payment",
		Source:      model.FeedbackSourceEmail,
		Status:      model.FeedbackStatusResolved,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	require.Equal(t, "payment question", list[0].Title)

	_, _, err = svc.List(context.Background(), dto.FeedbackListRequest{
		PageRequest: dto.PageRequest{Page: 1, PageSize: 20},
		StartTime:   "2026-13-01",
		EndTime:     "2026-01-02",
	})
	require.Error(t, err)
	appErr, ok := err.(*apperrors.AppError)
	require.True(t, ok)
	require.Equal(t, apperrors.ErrStartTimeFormat, appErr.Code)
}

func TestFeedbackServiceUpdateStatus(t *testing.T) {
	repo := &fakeFeedbackServiceRepo{items: map[string]*model.Feedback{
		"1": {
			BaseModel: model.BaseModel{ID: "1"},
			Source:    model.FeedbackSourceUser,
			Category:  "suggestion",
			Title:     "dark mode",
			Content:   "please add",
			Status:    model.FeedbackStatusOpen,
		},
	}}
	svc := &FeedbackService{repo: repo}

	updated, err := svc.UpdateStatus(context.Background(), "1", model.FeedbackStatusProcessing, "admin-1")
	require.NoError(t, err)
	require.Equal(t, model.FeedbackStatusProcessing, updated.Status)
	require.NotNil(t, updated.HandledBy)
	require.Equal(t, "admin-1", *updated.HandledBy)
	require.NotNil(t, updated.HandledAt)

	_, err = svc.UpdateStatus(context.Background(), "not-exist", model.FeedbackStatusClosed, "admin-1")
	require.Error(t, err)
	appErr, ok := err.(*apperrors.AppError)
	require.True(t, ok)
	require.Equal(t, apperrors.ErrFeedbackNotFound, appErr.Code)
}
