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
	Source         string        `gorm:"type:varchar(20);not null;index" json:"source"`                          // 反馈来源 (user/email)
	Category       string        `gorm:"type:varchar(64)" json:"category"`                                       // 反馈分类
	Title          string        `gorm:"type:varchar(255);not null" json:"title"`                                // 反馈标题
	Content        string        `gorm:"type:text;not null" json:"content"`                                      // 反馈详情内容
	UserID         *string       `gorm:"type:uuid;index" json:"user_id"`                                         // 用户ID (当来源为user时有效)
	User           *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`                                // 关联的用户对象
	Email          string        `gorm:"type:varchar(128)" json:"email"`                                         // 发件人邮箱 (当来源为email时有效)
	InboundEmailID *string       `gorm:"type:uuid;index" json:"inbound_email_id"`                                // 关联的同步邮件记录ID
	InboundEmail   *InboundEmail `gorm:"foreignKey:InboundEmailID" json:"inbound_email,omitempty"`               // 关联的同步邮件记录对象
	Status         string        `gorm:"type:varchar(20);not null;default:'open';index" json:"status"`           // 处理状态
	HandledBy      *string       `gorm:"type:uuid" json:"handled_by"`                                            // 处理人用户ID
	HandledAt      *time.Time    `json:"handled_at"`                                                             // 处理时间
}

// SortableFields 返回反馈列表查询时允许排序的字段列表
func (Feedback) SortableFields() []string {
	return []string{"created_at", "updated_at", "status", "source"}
}
