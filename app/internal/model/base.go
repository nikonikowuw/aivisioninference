package model

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// contextKey 是请求作用域值的上下文键类型（避免字符串键冲突）
type contextKey string

const (
	// ContextKeyUserID 仅用于 context.Context（context.WithValue / Value）
	// 不要与 middleware.ContextKeyUserID 混淆，后者用于 gin.Context.Set/Get
	ContextKeyUserID contextKey = "user_id"
)

// BaseModel 包含所有模型的公共字段
// 在模型中嵌入此结构体以获取标准审计字段
// CreatedBy 和 UpdatedBy 由 GORM 钩子从请求上下文中自动填充
type BaseModel struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"` // 记录的主键 ID (UUID)
	CreatedAt time.Time      `json:"created_at"`                                               // 记录创建时间
	UpdatedAt time.Time      `json:"updated_at"`                                               // 记录更新时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                                           // 软删除标记
	CreatedBy *string        `gorm:"type:uuid" json:"created_by"`                              // 创建记录的用户 ID
	UpdatedBy *string        `gorm:"type:uuid" json:"updated_by"`                              // 最后修改记录的用户 ID
}

// BeforeCreate 是 GORM 钩子，从请求上下文中自动填充 CreatedBy
// 仅在非请求上下文（种子数据、迁移、后台任务）中允许缺少 user_id
// 如果上下文存在但没有 user_id，会记录警告日志以提醒调用者
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if tx.Statement != nil && tx.Statement.Context != nil {
		if userID, ok := tx.Statement.Context.Value(ContextKeyUserID).(string); ok && userID != "" {
			b.CreatedBy = &userID
			return nil
		}
		zap.L().Warn("BaseModel.BeforeCreate: context exists but user_id not found — CreatedBy will be empty")
		return nil
	}
	// 在请求作用域外（如种子数据、迁移、后台任务）期望 nil 上下文
	return nil
}

// BeforeUpdate 是 GORM 钩子，从请求上下文中自动填充 UpdatedBy
// 仅在非请求上下文（种子数据、迁移、后台任务）中允许缺少 user_id
// 如果上下文存在但没有 user_id，会记录警告日志以提醒调用者
func (b *BaseModel) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement != nil && tx.Statement.Context != nil {
		if userID, ok := tx.Statement.Context.Value(ContextKeyUserID).(string); ok && userID != "" {
			b.UpdatedBy = &userID
			return nil
		}
		zap.L().Warn("BaseModel.BeforeUpdate: context exists but user_id not found — UpdatedBy will be empty")
		return nil
	}
	// 在请求作用域外（如种子数据、迁移、后台任务）期望 nil 上下文
	return nil
}

