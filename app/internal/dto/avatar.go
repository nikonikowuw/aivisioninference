// Package dto 定义请求和响应的数据传输结构体，包含参数校验和序列化标签。
package dto

// AvatarUploadResponse is the response for avatar upload.
type AvatarUploadResponse struct {
	AvatarURL string `json:"avatar_url"`
}
