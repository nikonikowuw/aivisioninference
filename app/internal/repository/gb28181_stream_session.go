package repository

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/model"
	"gorm.io/gorm"
)

type GB28181StreamSessionRepository struct {
	db *gorm.DB
}

func NewGB28181StreamSessionRepository(db *gorm.DB) *GB28181StreamSessionRepository {
	return &GB28181StreamSessionRepository{db: db}
}

func (r *GB28181StreamSessionRepository) Create(ctx context.Context, session *model.GB28181StreamSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *GB28181StreamSessionRepository) Update(ctx context.Context, streamID string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.GB28181StreamSession{}).Where("stream_id = ?", streamID).Updates(updates).Error
}

func (r *GB28181StreamSessionRepository) FindByStreamID(ctx context.Context, streamID string) (*model.GB28181StreamSession, error) {
	var session model.GB28181StreamSession
	if err := r.db.WithContext(ctx).Where("stream_id = ?", streamID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *GB28181StreamSessionRepository) FindByCallID(ctx context.Context, callID string) (*model.GB28181StreamSession, error) {
	var session model.GB28181StreamSession
	if err := r.db.WithContext(ctx).Where("sip_call_id = ?", callID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *GB28181StreamSessionRepository) Delete(ctx context.Context, streamID string) error {
	return r.db.WithContext(ctx).Where("stream_id = ?", streamID).Delete(&model.GB28181StreamSession{}).Error
}
