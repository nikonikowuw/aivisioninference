package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/i18n"
)

const maxAuditSummaryLength = 255

// skipPrefixes 列出无需审计记录的路径前缀，避免每次请求重复分配。
var skipPrefixes = []string{
	"/health",
	"/swagger",
	"/api/v1/auth/login",
	"/api/v1/auth/refresh",
	"/api/v1/audit-logs",
	"/static/",
}

// AuditLogger 定义审计日志写入能力，便于中间件解耦与测试注入。
type AuditLogger interface {
	Create(ctx context.Context, log *model.AuditLog) error
}

// Audit 在请求完成后统一写入审计日志，并保证审计失败不影响业务响应。
func Audit(svc AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil || shouldSkipAudit(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		status := inferAuditResponseStatus(c)
		auditLog := &model.AuditLog{
			ResourceType:   inferAuditResourceType(c.Request.URL.Path),
			RequestPath:    c.FullPath(),
			RequestMethod:  c.Request.Method,
			RequestIP:      c.ClientIP(),
			UserAgent:      TruncateAuditSummary(c.Request.UserAgent(), 512),
			ResponseStatus: status,
			DurationMs:     time.Since(start).Milliseconds(),
			ResultSummary:  TruncateAuditSummary(inferAuditResultSummary(status), maxAuditSummaryLength),
			ActionType:     inferAuditActionType(c.Request.Method, inferAuditResourceType(c.Request.URL.Path), c.Request.URL.Path),
		}
		if auditLog.RequestPath == "" {
			auditLog.RequestPath = c.Request.URL.Path
		}

		if userID, ok := c.Get(ContextKeyUserID); ok {
			if uid, ok := userID.(string); ok && uid != "" {
				auditLog.UserID = &uid
			}
		}
		if username, ok := c.Get(ContextKeyUsername); ok {
			if name, ok := username.(string); ok && name != "" {
				auditLog.Username = name
			}
		}

		if err := svc.Create(c.Request.Context(), auditLog); err != nil {
			zap.L().Warn("write audit log failed",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Error(err),
			)
		}
	}
}

// shouldSkipAudit 判断当前请求是否应跳过审计记录。
func shouldSkipAudit(method, path string) bool {
	if method == http.MethodOptions {
		return true
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// inferAuditResourceType 从请求路径提取资源类型，用于审计归类。
func inferAuditResourceType(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "v1" {
		return parts[2]
	}
	return parts[0]
}

// methodActions 定义 HTTP 方法对应的 i18n action key。
var methodActions = map[string]string{
	"GET":    "view",
	"POST":   "create",
	"PUT":    "update",
	"PATCH":  "update",
	"DELETE": "delete",
}

// inferAuditActionType 根据 HTTP 方法和资源类型生成 i18n key。
// 格式: action.{method}.{resource}
// 示例: action.create.users, action.view.roles
func inferAuditActionType(method, resourceType, path string) string {
	action, ok := methodActions[method]
	if !ok {
		action = "operate"
	}

	// 特殊路径处理
	if resourceType == "auth" {
		if method == "POST" {
			// 区分登录和登出
			if strings.Contains(path, "/logout") {
				return i18n.ActionLogout
			}
			return i18n.ActionLogin
		}
		return i18n.ActionAuth
	}

	return "action." + action + "." + resourceType
}

// inferAuditResponseStatus 推断审计应记录的最终响应状态码。
func inferAuditResponseStatus(c *gin.Context) int {
	if c == nil {
		return http.StatusOK
	}
	if len(c.Errors) == 0 {
		return c.Writer.Status()
	}
	lastErr := c.Errors.Last()
	if lastErr == nil || lastErr.Err == nil {
		return c.Writer.Status()
	}
	var appErr *apperrors.AppError
	if errors.As(lastErr.Err, &appErr) {
		return AuditHTTPStatusFromCode(appErr.Code)
	}
	return http.StatusInternalServerError
}

// AuditHTTPStatusFromCode 按项目错误码区间映射 HTTP 状态码。
func AuditHTTPStatusFromCode(code int) int {
	switch {
	case code >= 10000 && code < 20000:
		return http.StatusBadRequest
	case code >= 20000 && code < 30000:
		return http.StatusUnauthorized
	case code >= 30000 && code < 40000:
		return http.StatusForbidden
	case code >= 40000 && code < 50000:
		return http.StatusNotFound
	case code >= 50000:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// inferAuditResultSummary 根据 HTTP 状态码生成安全结果摘要。
// 返回 "success"（2xx/3xx）或 "failed"（其他），不带具体错误类型，
// 避免泄露底层细节。如需按错误类型筛选，可通过 response_status 字段实现。
func inferAuditResultSummary(status int) string {
	if status >= 200 && status < 400 {
		return "success"
	}
	return "failed"
}

// TruncateAuditSummary 将审计摘要按字符数截断到数据库字段允许长度。
func TruncateAuditSummary(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
