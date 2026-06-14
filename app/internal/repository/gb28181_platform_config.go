package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

type GB28181PlatformConfigRepository struct {
	db *gorm.DB
}

func NewGB28181PlatformConfigRepository(db *gorm.DB) *GB28181PlatformConfigRepository {
	return &GB28181PlatformConfigRepository{db: db}
}

func (r *GB28181PlatformConfigRepository) Get(ctx context.Context) (*model.GB28181PlatformConfig, error) {
	var cfg model.GB28181PlatformConfig
	err := r.db.WithContext(ctx).Where("id = ?", "default").First(&cfg).Error
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *GB28181PlatformConfigRepository) Update(ctx context.Context, cfg *model.GB28181PlatformConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}
