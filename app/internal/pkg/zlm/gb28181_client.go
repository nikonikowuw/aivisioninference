package zlm

import (
	"context"
	"net/http"
	"net/url"
)

// ====== GB28181 RTP 接收端（设备向平台推流） ======

// OpenRtpServer 创建 GB28181 RTP 接收端口
func (c *Client) OpenRtpServer(ctx context.Context, req OpenRtpServerRequest) (int, error) {
	var resp OpenRtpServerResponse
	if err := c.doRequest(ctx, http.MethodPost, "openRtpServer", nil, req, &resp); err != nil {
		return 0, err
	}
	return resp.Port, nil
}

// CloseRtpServer 关闭 GB28181 RTP 接收端口
func (c *Client) CloseRtpServer(ctx context.Context, streamID string) error {
	return c.doRequest(ctx, http.MethodPost, "closeRtpServer", nil, CloseRtpServerRequest{
		StreamID: streamID,
	}, nil)
}

// ListRtpServer 列出所有 RTP 接收端口
func (c *Client) ListRtpServer(ctx context.Context) ([]RtpServerInfo, error) {
	var resp ListRtpServerResponse
	if err := c.doRequest(ctx, http.MethodGet, "listRtpServer", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// UpdateRtpServerSSRC 更新 RTP 服务器过滤的 SSRC
func (c *Client) UpdateRtpServerSSRC(ctx context.Context, streamID, ssrc string) error {
	return c.doRequest(ctx, http.MethodPost, "updateRtpServerSSRC", nil, UpdateRtpServerSSRCRequest{
		StreamID: streamID,
		SSRC:     ssrc,
	}, nil)
}

// ====== GB28181 PS-RTP 发送端（平台向设备推流） ======

// StartSendRtp 启动 PS-RTP 推流（GB28181 客户端模式）
func (c *Client) StartSendRtp(ctx context.Context, req StartSendRtpRequest) (int, error) {
	var resp StartSendRtpResponse
	if err := c.doRequest(ctx, http.MethodPost, "startSendRtp", nil, req, &resp); err != nil {
		return 0, err
	}
	return resp.LocalPort, nil
}

// StopSendRtp 停止 PS-RTP 推流
func (c *Client) StopSendRtp(ctx context.Context, req StopSendRtpRequest) error {
	return c.doRequest(ctx, http.MethodPost, "stopSendRtp", nil, req, nil)
}

// ListRtpSender 列出所有 RTP 发送者
func (c *Client) ListRtpSender(ctx context.Context, vhost, app, stream string) ([]RtpSenderInfo, error) {
	params := url.Values{}
	params.Set("vhost", vhost)
	params.Set("app", app)
	params.Set("stream", stream)
	var resp ListRtpSenderResponse
	if err := c.doRequest(ctx, http.MethodGet, "listRtpSender", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// ====== 录像回放控制 ======

// SetRecordSpeed 设置录像回放倍速
func (c *Client) SetRecordSpeed(ctx context.Context, req SetRecordSpeedRequest) error {
	return c.doRequest(ctx, http.MethodPost, "setRecordSpeed", nil, req, nil)
}

// SeekRecordStamp 录像跳转到指定时间戳
func (c *Client) SeekRecordStamp(ctx context.Context, req SeekRecordStampRequest) error {
	return c.doRequest(ctx, http.MethodPost, "seekRecordStamp", nil, req, nil)
}

// ====== GB28181 SIP MESSAGE 发送（通过 ZLM 内置 SIP） ======

// 注：ZLM 没有专门的 sendSipMessage API，GB28181 SIP MESSAGE 由 ZLM 内部 SIP 栈处理。
// Go 端通过 /zlm/callback/on_catalog、/zlm/callback/on_alarm 等 webhook 接收响应。
