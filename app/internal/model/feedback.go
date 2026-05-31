// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

import "time"

// 反馈来源和状态常量定义
const (
	FeedbackSourceUser  = "user"  // 来源：系统用户手动提交
	FeedbackSourceEmail = "email" // 来源：邮件同步自动生成

	FeedbackStatusOpen       = "open"       // 状态：待处理/开启
	FeedbackStatusProcessing = "processing" // 状态：处理中
	FeedbackStatusResolved   = "resolved"   // 状态：已解决
	FeedbackStatusClosed     = "closed"     // 状态：已关闭
)

// Feedback 存储系统中提交的反馈或从邮件同步的反馈
type Feedback struct {
	BaseModel
	Source         string        `gorm:"type:varchar(20);not null;index" json:"source"`                // (user/email)
	Category       string        `gorm:"type:varchar(64)" json:"category"`
	Title          string        `gorm:"type:varchar(255);not null" json:"title"`
	Content        string        `gorm:"type:text;not null" json:"content"`
	UserID         *string       `gorm:"type:uuid;index" json:"user_id"`                               // (当来源为user时有效)
	User           *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Email          string        `gorm:"type:varchar(128)" json:"email"`                               // (当来源为email时有效)
	InboundEmailID *string       `gorm:"type:uuid;index" json:"inbound_email_id"`
	InboundEmail   *InboundEmail `gorm:"foreignKey:InboundEmailID" json:"inbound_email,omitempty"`
	Status         string        `gorm:"type:varchar(20);not null;default:'open';index" json:"status"`
	HandledBy      *string       `gorm:"type:uuid" json:"handled_by"`
	HandledAt      *time.Time    `json:"handled_at"`
}

// SortableFields 返回反馈列表查询时允许排序的字段列表
func (Feedback) SortableFields() []string {
	return []string{"created_at", "updated_at", "status", "source"}
}
