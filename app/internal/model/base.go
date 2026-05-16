package model

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Context key type for request-scoped values (avoids string key collisions).
type contextKey string

const (
	// ContextKeyUserID is used only with context.Context (context.WithValue / Value).
	// Do not mix it with middleware.ContextKeyUserID, which is for gin.Context.Set/Get.
	ContextKeyUserID contextKey = "user_id"
)

// BaseModel contains common fields for all models.
// Embed this struct in your models to get standard audit fields.
// CreatedBy and UpdatedBy are auto-populated by GORM hooks from request context.
type BaseModel struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy string         `gorm:"type:uuid" json:"created_by"`
	UpdatedBy string         `gorm:"type:uuid" json:"updated_by"`
}

// BeforeCreate is a GORM hook that auto-populates CreatedBy from request context.
// A missing user_id in context is allowed only in non-request contexts (seeds, migrations, background jobs).
// If the context exists but has no user_id, a warning is logged to alert the caller.
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if tx.Statement != nil && tx.Statement.Context != nil {
		if userID, ok := tx.Statement.Context.Value(ContextKeyUserID).(string); ok && userID != "" {
			b.CreatedBy = userID
			return nil
		}
		zap.L().Warn("BaseModel.BeforeCreate: context exists but user_id not found — CreatedBy will be empty")
		return nil
	}
	// nil context is expected outside request scope (e.g. seeds, migrations, background jobs)
	return nil
}

// BeforeUpdate is a GORM hook that auto-populates UpdatedBy from request context.
// A missing user_id in context is allowed only in non-request contexts (seeds, migrations, background jobs).
// If the context exists but has no user_id, a warning is logged to alert the caller.
func (b *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement != nil && tx.Statement.Context != nil {
		if userID, ok := tx.Statement.Context.Value(ContextKeyUserID).(string); ok && userID != "" {
			b.UpdatedBy = userID
			return nil
		}
		zap.L().Warn("BaseModel.BeforeUpdate: context exists but user_id not found — UpdatedBy will be empty")
		return nil
	}
	// nil context is expected outside request scope (e.g. seeds, migrations, background jobs)
	return nil
}
