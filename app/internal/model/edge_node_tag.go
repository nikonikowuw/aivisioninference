// Package model 定义系统数据模型，包含 GORM 结构体和数据库表映射。
package model

// EdgeNodeTag 边缘节点标签
type EdgeNodeTag struct {
	BaseModel
	Name        string     `gorm:"type:varchar(100);not null;uniqueIndex;comment:标签名称" json:"name"`
	Color       string     `gorm:"type:varchar(7);default:'#1890ff';comment:标签颜色(Hex)" json:"color"`
	Description string     `gorm:"type:varchar(500);comment:标签描述" json:"description"`
	EdgeNodes   []EdgeNode `gorm:"many2many:edge_node_tag_relations;" json:"edge_nodes,omitempty"`
}

// TableName 指定表名
func (EdgeNodeTag) TableName() string {
	return "edge_node_tags"
}

// SortableFields 返回允许排序的字段列表
func (EdgeNodeTag) SortableFields() []string {
	return []string{"created_at", "name"}
}

// EdgeNodeTagRelation 边缘节点与标签多对多关联表
type EdgeNodeTagRelation struct {
	EdgeNodeID    string `gorm:"type:uuid;primaryKey;comment:边缘节点ID" json:"edge_node_id"`
	EdgeNodeTagID string `gorm:"type:uuid;primaryKey;comment:标签ID" json:"edge_node_tag_id"`
}

// TableName 指定表名
func (EdgeNodeTagRelation) TableName() string {
	return "edge_node_tag_relations"
}
