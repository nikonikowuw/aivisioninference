package model

import "time"

const (
	FeedbackSourceUser  = "user"
	FeedbackSourceEmail = "email"

	FeedbackStatusOpen       = "open"
	FeedbackStatusProcessing = "processing"
	FeedbackStatusResolved   = "resolved"
	FeedbackStatusClosed     = "closed"
)

// Feedback 存储系统中提交的反馈或从邮件同步的反馈
type Feedback struct {
	BaseModel
	Source         string        `gorm:"type:varchar(20);not null;index" json:"source"`
	Category       string        `gorm:"type:varchar(64)" json:"category"`
	Title          string        `gorm:"type:varchar(255);not null" json:"title"`
	Content        string        `gorm:"type:text;not null" json:"content"`
	UserID         *string       `gorm:"type:uuid;index" json:"user_id"`
	User           *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Email          string        `gorm:"type:varchar(128)" json:"email"`
	InboundEmailID *string       `gorm:"type:uuid;index" json:"inbound_email_id"`
	InboundEmail   *InboundEmail `gorm:"foreignKey:InboundEmailID" json:"inbound_email,omitempty"`
	Status         string        `gorm:"type:varchar(20);not null;default:'open';index" json:"status"`
	HandledBy      *string       `gorm:"type:uuid" json:"handled_by"`
	HandledAt      *time.Time    `json:"handled_at"`
}

// SortableFields 返回允许排序的字段列表
func (Feedback) SortableFields() []string {
	return []string{"created_at", "updated_at", "status", "source"}
}
