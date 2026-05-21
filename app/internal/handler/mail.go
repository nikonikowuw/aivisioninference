package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

// MailHandler handles system mail configuration requests.
type MailHandler struct {
	svc *service.MailService
}

func NewMailHandler(svc *service.MailService) *MailHandler {
	return &MailHandler{svc: svc}
}

// GetConfig returns sanitized mail settings.
//
// @Summary      获取邮件配置
// @Tags         系统配置
// @Produce      json
// @Success      200  {object}  dto.Response{data=dto.MailConfigResponse}
// @Router       /system/mail-config [get]
// @Security     BearerAuth
func (h *MailHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig(c.Request.Context())
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, cfg)
}

// SaveConfig saves mail settings.
//
// @Summary      保存邮件配置
// @Tags         系统配置
// @Accept       json
// @Produce      json
// @Param        body  body  dto.MailConfigRequest  true  "邮件配置"
// @Success      200   {object}  dto.Response{data=dto.MailConfigResponse}
// @Router       /system/mail-config [put]
// @Security     BearerAuth
func (h *MailHandler) SaveConfig(c *gin.Context) {
	var req dto.MailConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	cfg, err := h.svc.SaveConfig(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, cfg)
}

// TestSMTP sends a test email.
//
// @Summary      测试 SMTP 发信
// @Tags         系统配置
// @Accept       json
// @Produce      json
// @Param        body  body  dto.TestSMTPRequest  true  "测试收件人"
// @Success      200   {object}  dto.Response
// @Router       /system/mail-config/test-smtp [post]
// @Security     BearerAuth
func (h *MailHandler) TestSMTP(c *gin.Context) {
	var req dto.TestSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	if err := h.svc.TestSMTP(c.Request.Context(), req.To); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// TestIMAP tests IMAP connection.
//
// @Summary      测试 IMAP 连接
// @Tags         系统配置
// @Produce      json
// @Success      200  {object}  dto.Response
// @Router       /system/mail-config/test-imap [post]
// @Security     BearerAuth
func (h *MailHandler) TestIMAP(c *gin.Context) {
	if err := h.svc.TestIMAP(c.Request.Context()); err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, nil)
}

// SyncIMAP synchronizes feedback emails.
//
// @Summary      同步反馈邮件
// @Tags         系统配置
// @Produce      json
// @Success      200  {object}  dto.Response
// @Router       /system/mail-config/sync-imap [post]
// @Security     BearerAuth
func (h *MailHandler) SyncIMAP(c *gin.Context) {
	count, err := h.svc.SyncIMAP(c.Request.Context(), 20)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, gin.H{"synced": count})
}

// FeedbackHandler handles user feedback requests.
type FeedbackHandler struct {
	svc *service.FeedbackService
}

func NewFeedbackHandler(svc *service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{svc: svc}
}

// Create submits user feedback.
//
// @Summary      提交反馈
// @Tags         用户反馈
// @Accept       json
// @Produce      json
// @Param        body  body  dto.FeedbackCreateRequest  true  "反馈内容"
// @Success      200   {object}  dto.Response
// @Router       /feedback [post]
// @Security     BearerAuth
func (h *FeedbackHandler) Create(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req dto.FeedbackCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	item, err := h.svc.Create(c.Request.Context(), uid, req)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}

// List returns feedback list.
//
// @Summary      反馈列表
// @Tags         用户反馈
// @Produce      json
// @Success      200  {object}  dto.Response
// @Router       /feedback [get]
// @Security     BearerAuth
func (h *FeedbackHandler) List(c *gin.Context) {
	var req dto.FeedbackListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		attachError(c, err)
		return
	}
	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// UpdateStatus updates feedback status.
//
// @Summary      更新反馈状态
// @Tags         用户反馈
// @Accept       json
// @Produce      json
// @Param        id    path  string                           true  "反馈 ID"
// @Param        body  body  dto.FeedbackUpdateStatusRequest true  "状态"
// @Success      200   {object}  dto.Response
// @Router       /feedback/{id}/status [put]
// @Security     BearerAuth
func (h *FeedbackHandler) UpdateStatus(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req dto.FeedbackUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		attachError(c, badRequestError(c, err))
		return
	}
	item, err := h.svc.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status, uid)
	if err != nil {
		attachError(c, err)
		return
	}
	response.OK(c, item)
}
