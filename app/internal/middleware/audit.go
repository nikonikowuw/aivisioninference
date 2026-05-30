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

		// 在 c.Next() 之前记录开始时间，之后获取 DurationMs，精确测量请求处理耗时。
		start := time.Now()
		c.Next()

		// 通过四个独立的推断函数组合审计日志，每个函数只负责一个维度的数据，职责单一。
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
		// FullPath 返回注册的路由（如 /api/v1/users/:id），比原始 URL 路径更适合审计归类。
		// 如果路由未匹配到，Fallback 到原始 URL 路径。
		if auditLog.RequestPath == "" {
			auditLog.RequestPath = c.Request.URL.Path
		}

		// 从 Auth 中间件设置的 Gin Context 中获取用户信息，关联审计记录到具体用户。
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

		// 审计写入失败仅记录警告日志，不阻塞业务响应。
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
// 默认路由格式为 /api/v1/{resource}/...，从中提取第3段作为资源类型。
// 非标准路径则取第一个 path segment。
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
// 特殊路径会单独处理，生成语义更准确的 key。
func inferAuditActionType(method, resourceType, path string) string {
	action, ok := methodActions[method]
	if !ok {
		action = "operate"
	}

	// 特殊路径处理
	switch resourceType {
	case "auth":
		return inferAuthAction(method, path)
	case "tasks":
		// 取消任务不应归类为 create
		if method == "POST" && strings.Contains(path, "/cancel") {
			return i18n.ActionCancelTasks
		}
	case "brand-config":
		// 上传品牌 Logo 不应归类为 create brand-config
		if strings.Contains(path, "/logo") {
			return i18n.ActionUploadBrandLogo
		}
	case "feedback":
		// 用户提交反馈不应归类为 create（非管理员）
		if method == "POST" {
			return i18n.ActionCreateFeedback
		}
	}

	return "action." + action + "." + resourceType
}

// inferAuthAction 根据 auth 子路径推断更精确的操作类型 key。
// 避免所有 auth 操作都被笼统地归类为 login 或 auth。
func inferAuthAction(method, path string) string {
	switch {
	case strings.Contains(path, "/logout"):
		return i18n.ActionLogout
	case strings.Contains(path, "/password-reset"):
		return i18n.ActionPasswordReset
	case strings.Contains(path, "/password") && method == "PUT":
		return i18n.ActionChangePassword
	case strings.Contains(path, "/profile") && method == "PUT":
		return i18n.ActionUpdateProfile
	case strings.Contains(path, "/avatar") && method == "POST":
		return i18n.ActionUploadAvatar
	case strings.Contains(path, "/me") && method == "GET":
		return i18n.ActionViewProfile
	case method == "POST":
		return i18n.ActionLogin
	default:
		return i18n.ActionLogin
	}
}

// inferAuditResponseStatus 推断审计应记录的最终响应状态码。
// 优先从 Gin 错误链中提取业务错误码映射 HTTP 状态码，否则使用实际响应的状态码。
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
	// 使用 errors.As 而非类型断言，因为错误可能被多层包装（如 fmt.Errorf("...: %w", err)）。
	var appErr *apperrors.AppError
	if errors.As(lastErr.Err, &appErr) {
		return AuditHTTPStatusFromCode(appErr.Code)
	}
	return http.StatusInternalServerError
}

// AuditHTTPStatusFromCode 按项目错误码区间映射 HTTP 状态码。
// 错误码区间规则：1xxxx=参数错误, 2xxxx=认证失败, 3xxxx=权限不足, 4xxxx=资源未找到, 5xxxx=系统异常。
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
