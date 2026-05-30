// Package repository 提供数据访问层实现，封装 GORM 数据库操作。
package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

// BrandConfigRepository 处理系统品牌配置的数据库读写操作。
type BrandConfigRepository struct {
	db *gorm.DB
}

// NewBrandConfigRepository 创建并返回一个新的 BrandConfigRepository 实例。
func NewBrandConfigRepository(db *gorm.DB) *BrandConfigRepository {
	return &BrandConfigRepository{db: db}
}

// First 获取数据库中创建时间最早的第一条系统品牌配置。
func (r *BrandConfigRepository) First(ctx context.Context) (*model.BrandConfig, error) {
	var cfg model.BrandConfig
	err := r.db.WithContext(ctx).Order("created_at ASC").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &cfg, err
}

// Save 新增或保存品牌配置的修改。
func (r *BrandConfigRepository) Save(ctx context.Context, cfg *model.BrandConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}
