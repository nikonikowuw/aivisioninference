package dto

// ImportDeviceRequest 设备导入请求（单个设备从暂存区导入到正式设备列表）
type ImportDeviceRequest struct {
	// 设备名称（可选，为空时使用扫描到的默认名称）
	DeviceName string `json:"device_name" validate:"omitempty,max=128"`
	// 摄像头登录用户名
	Username string `json:"username" validate:"required,max=128"`
	// 摄像头登录密码
	Password string `json:"password" validate:"required,max=256"`
	// 是否启用自动推理（默认 true）
	EnableInfer *bool `json:"enable_infer" validate:"omitempty"`
	// 设备分组 ID（可选）
	GroupID *string `json:"group_id" validate:"omitempty,uuid"`
}

// BatchImportDeviceRequest 批量设备导入请求
type BatchImportDeviceRequest struct {
	// 待导入的设备 ID 列表
	IDs []string `json:"ids" validate:"required,min=1"`
	// 摄像头登录用户名（批量导入使用相同凭证）
	Username string `json:"username" validate:"required,max=128"`
	// 摄像头登录密码（批量导入使用相同凭证）
	Password string `json:"password" validate:"required,max=256"`
	// 是否启用自动推理（默认 true）
	EnableInfer *bool `json:"enable_infer" validate:"omitempty"`
}
