package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
)

type MediaCapacityReservationRepository struct {
	db *gorm.DB
}

func NewMediaCapacityReservationRepository(db *gorm.DB) *MediaCapacityReservationRepository {
	return &MediaCapacityReservationRepository{db: db}
}

func (r *MediaCapacityReservationRepository) WithTx(tx *gorm.DB) *MediaCapacityReservationRepository {
	return &MediaCapacityReservationRepository{db: tx}
}

func (r *MediaCapacityReservationRepository) Create(ctx context.Context, item *model.MediaCapacityReservation) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *MediaCapacityReservationRepository) FindValidByStreamKey(ctx context.Context, streamKey string, now time.Time) (*model.MediaCapacityReservation, error) {
	var item model.MediaCapacityReservation
	err := r.db.WithContext(ctx).
		Where("stream_key = ?", streamKey).
		Where("status = ? OR lease_expires > ?", model.MediaReservationConfirmed, now).
		First(&item).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *MediaCapacityReservationRepository) DeleteExpired(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("status = ? AND lease_expires IS NOT NULL AND lease_expires <= ?", model.MediaReservationPending, now).
		Delete(&model.MediaCapacityReservation{}).Error
}

func (r *MediaCapacityReservationRepository) ListEffective(ctx context.Context, now time.Time) ([]model.MediaCapacityReservation, error) {
	var items []model.MediaCapacityReservation
	err := r.db.WithContext(ctx).
		Where("status = ? OR lease_expires > ?", model.MediaReservationConfirmed, now).
		Find(&items).Error
	return items, err
}

func (r *MediaCapacityReservationRepository) Confirm(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.MediaCapacityReservation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"status": model.MediaReservationConfirmed, "lease_expires": nil}).Error
}

func (r *MediaCapacityReservationRepository) Release(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&model.MediaCapacityReservation{}).Error
}
