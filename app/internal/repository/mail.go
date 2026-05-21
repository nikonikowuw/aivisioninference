package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// MailConfigRepository handles database operations for mail settings.
type MailConfigRepository struct {
	db *gorm.DB
}

func NewMailConfigRepository(db *gorm.DB) *MailConfigRepository {
	return &MailConfigRepository{db: db}
}

func (r *MailConfigRepository) First(ctx context.Context) (*model.MailConfig, error) {
	var cfg model.MailConfig
	err := r.db.WithContext(ctx).Order("created_at ASC").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}

func (r *MailConfigRepository) Save(ctx context.Context, cfg *model.MailConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}

// EmailTokenRepository handles database operations for email tokens.
type EmailTokenRepository struct {
	db *gorm.DB
}

func NewEmailTokenRepository(db *gorm.DB) *EmailTokenRepository {
	return &EmailTokenRepository{db: db}
}

func (r *EmailTokenRepository) Create(ctx context.Context, item *model.EmailToken) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *EmailTokenRepository) FindByHash(ctx context.Context, tokenHash, purpose string) (*model.EmailToken, error) {
	var item model.EmailToken
	err := r.db.WithContext(ctx).Where("token_hash = ? AND purpose = ?", tokenHash, purpose).First(&item).Error
	return &item, err
}

func (r *EmailTokenRepository) CountSince(ctx context.Context, email, purpose, requestIP string, since time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.EmailToken{}).Where("email = ? AND purpose = ?", email, purpose)
	if requestIP != "" {
		query = query.Where("request_ip = ?", requestIP)
	}
	err := query.Where("created_at > ?", since).Count(&count).Error
	return count, err
}

func (r *EmailTokenRepository) Update(ctx context.Context, item *model.EmailToken) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// InboundEmailRepository handles database operations for synchronized emails.
type InboundEmailRepository struct {
	db *gorm.DB
}

func NewInboundEmailRepository(db *gorm.DB) *InboundEmailRepository {
	return &InboundEmailRepository{db: db}
}

func (r *InboundEmailRepository) Exists(ctx context.Context, account, mailbox string, uid uint32, messageID string) (bool, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.InboundEmail{}).Where("account = ? AND mailbox = ?", account, mailbox)
	if uid > 0 {
		query = query.Where("uid = ?", uid)
	} else if messageID != "" {
		query = query.Where("message_id = ?", messageID)
	} else {
		return false, nil
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *InboundEmailRepository) Create(ctx context.Context, item *model.InboundEmail) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FeedbackRepository handles database operations for feedback.
type FeedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func (r *FeedbackRepository) Create(ctx context.Context, item *model.Feedback) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *FeedbackRepository) FindByID(ctx context.Context, id string) (*model.Feedback, error) {
	var item model.Feedback
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

func (r *FeedbackRepository) Update(ctx context.Context, item *model.Feedback) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *FeedbackRepository) List(ctx context.Context, req dto.FeedbackListRequest) ([]model.Feedback, int64, error) {
	var items []model.Feedback
	var total int64
	query := r.db.WithContext(ctx).Model(&model.Feedback{}).Scopes(req.FilterScopes()...)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Scopes(
		scopes.Paginate(req.GetPage(), req.GetPageSize()),
		scopes.OrderBy(req.Sort, req.Order, model.Feedback{}.SortableFields()...),
	).Preload("User").Preload("InboundEmail").Find(&items).Error
	return items, total, err
}
