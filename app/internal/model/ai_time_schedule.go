package model

import "gorm.io/datatypes"

// AITimeSchedule AI推理时间配置模板
// 可被多个 AIVisionTask 引用，避免重复配置相同的日期与时间窗。
type AITimeSchedule struct {
	BaseModel
	Name        string         `gorm:"type:varchar(255);not null;comment:配置名称" json:"name"`
	Description string         `gorm:"type:varchar(500);comment:描述" json:"description"`
	StartDate   datatypes.Date `gorm:"type:date;not null;comment:生效起始日期" json:"start_date" swaggertype:"string"`
	EndDate     datatypes.Date `gorm:"type:date;not null;comment:生效结束日期" json:"end_date" swaggertype:"string"`
	TimeWindows datatypes.JSON `gorm:"type:jsonb;not null;comment:每天运行的时间段 [{start,end}]" json:"time_windows" swaggertype:"object"`
}

// TableName 指定表名
func (AITimeSchedule) TableName() string {
	return "ai_time_schedules"
}

// SortableFields 返回允许排序的字段列表
func (AITimeSchedule) SortableFields() []string {
	return []string{"created_at", "updated_at", "name"}
}
