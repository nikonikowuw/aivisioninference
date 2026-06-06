package zlm

// ====== GB28181 RTP 服务器（接收） ======

// OpenRtpServerRequest 创建 GB28181 RTP 接收端口
type OpenRtpServerRequest struct {
	Port     int    `json:"port"`      // 0 则随机
	TCPMode  int    `json:"tcp_mode"`  // 0:UDP, 1:TCP被动, 2:TCP主动
	StreamID string `json:"stream_id"` // 绑定的流 ID
}

// OpenRtpServerResponse 创建 RTP 服务器返回
type OpenRtpServerResponse struct {
	ZLMRsp
	Port int `json:"port"`
}

// CloseRtpServerRequest 关闭 GB28181 RTP 接收端口
type CloseRtpServerRequest struct {
	StreamID string `json:"stream_id"`
}

// RtpServerInfo RTP 服务器信息
type RtpServerInfo struct {
	Port     int    `json:"port"`
	StreamID string `json:"stream_id"`
}

// ListRtpServerResponse RTP 服务器列表
type ListRtpServerResponse struct {
	ZLMRsp
	Data []RtpServerInfo `json:"data"`
}

// ====== GB28181 PS-RTP 推流（发送） ======

// StartSendRtpRequest 启动 PS-RTP 推流
// 必填字段：Vhost, App, Stream, SSRC, DstURL, DstPort, IsUDP
// 可选字段：SrcPort, PT, UsePS, OnlyAudio
//
// JSON key 通过 tag 显式指定，字段顺序不影响序列化。
// 此处按逻辑分组排列以提升可读性：流信息 → 目标 → 源/可选参数。
type StartSendRtpRequest struct {
	// 流信息
	Vhost  string `json:"vhost"`
	App    string `json:"app"`
	Stream string `json:"stream"`
	SSRC   string `json:"ssrc"` // 16 进制字符串

	// 目标地址
	DstURL  string `json:"dst_url"`  // 目标 IP/域名
	DstPort int    `json:"dst_port"` // 目标端口
	IsUDP   int    `json:"is_udp"`   // 1:UDP, 0:TCP

	// 可选参数
	SrcPort   int `json:"src_port,omitempty"`
	PT        int `json:"pt,omitempty"`         // RTP payload type (uint8_t)
	UsePS     int `json:"use_ps,omitempty"`
	OnlyAudio int `json:"only_audio,omitempty"`
}

// StartSendRtpResponse 启动推流返回
type StartSendRtpResponse struct {
	ZLMRsp
	LocalPort int `json:"local_port"`
}

// StopSendRtpRequest 停止 PS-RTP 推流
type StopSendRtpRequest struct {
	Vhost  string `json:"vhost"`
	App    string `json:"app"`
	Stream string `json:"stream"`
	SSRC   string `json:"ssrc,omitempty"`
}

// RtpSenderInfo RTP 发送者信息
type RtpSenderInfo struct {
	SSRC     string `json:"ssrc"`
	DstURL   string `json:"dst_url"`
	DstPort  int    `json:"dst_port"`
	LocalIP  string `json:"local_ip"`
	LocalPort int   `json:"local_port"`
	IsUDP    int    `json:"is_udp"`
}

// ListRtpSenderResponse RTP 发送列表
type ListRtpSenderResponse struct {
	ZLMRsp
	Data []RtpSenderInfo `json:"data"`
}

// ====== 录像回放控制 ======

// SetRecordSpeedRequest 设置录像回放倍速
type SetRecordSpeedRequest struct {
	Vhost  string  `json:"vhost"`
	App    string  `json:"app"`
	Stream string  `json:"stream"`
	Speed  float64 `json:"speed"`
}

// SeekRecordStampRequest 录像跳转到指定时间戳
type SeekRecordStampRequest struct {
	Vhost  string `json:"vhost"`
	App    string `json:"app"`
	Stream string `json:"stream"`
	Stamp  int64  `json:"stamp"` // 毫秒
}

// UpdateRtpServerSSRCRequest 更新 RTP 服务器 SSRC
type UpdateRtpServerSSRCRequest struct {
	StreamID string `json:"stream_id"`
	SSRC     string `json:"ssrc"`
}
