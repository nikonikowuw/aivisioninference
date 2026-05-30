// Package repository 提供数据访问层实现，封装 GORM 数据库操作。
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

// MailConfigRepository 处理系统邮件配置的数据库读写操作
type MailConfigRepository struct {
	db *gorm.DB
}

// NewMailConfigRepository 创建并返回一个新的 MailConfigRepository 实例
func NewMailConfigRepository(db *gorm.DB) *MailConfigRepository {
	return &MailConfigRepository{db: db}
}

// First 获取数据库中创建时间最早的第一条系统邮件配置 (通常全局只有一条)
func (r *MailConfigRepository) First(ctx context.Context) (*model.MailConfig, error) {
	var cfg model.MailConfig
	err := r.db.WithContext(ctx).Order("created_at ASC").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}

// Save 新增或保存邮件配置的修改
func (r *MailConfigRepository) Save(ctx context.Context, cfg *model.MailConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}

// EmailTokenRepository 处理邮箱一次性校验验证码/令牌的数据库读写操作
type EmailTokenRepository struct {
	db *gorm.DB
}

// NewEmailTokenRepository 创建并返回一个新的 EmailTokenRepository 实例
func NewEmailTokenRepository(db *gorm.DB) *EmailTokenRepository {
	return &EmailTokenRepository{db: db}
}

// Create 写入一条新的邮箱校验令牌记录
func (r *EmailTokenRepository) Create(ctx context.Context, item *model.EmailToken) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByHash 根据 Token 的散列哈希值及具体用途类型查询对应的验证记录
func (r *EmailTokenRepository) FindByHash(ctx context.Context, tokenHash, purpose string) (*model.EmailToken, error) {
	var item model.EmailToken
	err := r.db.WithContext(ctx).Where("token_hash = ? AND purpose = ?", tokenHash, purpose).First(&item).Error
	return &item, err
}

// CountSince 统计自某个时间点以来，指定邮箱、用途和 IP 地址所发起验证请求的次数 (用于频次风控限制)
func (r *EmailTokenRepository) CountSince(ctx context.Context, email, purpose, requestIP string, since time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.EmailToken{}).Where("email = ? AND purpose = ?", email, purpose)
	if requestIP != "" {
		query = query.Where("request_ip = ?", requestIP)
	}
	err := query.Where("created_at > ?", since).Count(&count).Error
	return count, err
}

// Update 更新一条邮箱验证令牌的状态 (如: 标记已使用、增加尝试次数等)
func (r *EmailTokenRepository) Update(ctx context.Context, item *model.EmailToken) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// InboundEmailRepository 处理同步的入站邮件记录的数据库操作
type InboundEmailRepository struct {
	db *gorm.DB
}

// NewInboundEmailRepository 创建并返回一个新的 InboundEmailRepository 实例
func NewInboundEmailRepository(db *gorm.DB) *InboundEmailRepository {
	return &InboundEmailRepository{db: db}
}

// Exists 检查指定的邮件是否已被同步过，避免重复拉取
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

// Create 写入一条新的入站邮件记录
func (r *InboundEmailRepository) Create(ctx context.Context, item *model.InboundEmail) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FeedbackRepository 处理用户反馈相关的数据库操作
type FeedbackRepository struct {
	db *gorm.DB
}

// NewFeedbackRepository 创建并返回一个新的 FeedbackRepository 实例
func NewFeedbackRepository(db *gorm.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

// Create 写入一条新的用户反馈记录
func (r *FeedbackRepository) Create(ctx context.Context, item *model.Feedback) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByID 根据主键 ID 查询单个用户反馈的详细记录
func (r *FeedbackRepository) FindByID(ctx context.Context, id string) (*model.Feedback, error) {
	var item model.Feedback
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	return &item, err
}

// Update 更新反馈的处理状态与处理进度信息
func (r *FeedbackRepository) Update(ctx context.Context, item *model.Feedback) error {
	return r.db.WithContext(ctx).Save(item).Error
}

// List 分页且附带关联关系 (预加载 User 和 InboundEmail) 查询反馈列表
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
