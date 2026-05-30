package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTranslateAction 验证操作类型 key 的翻译逻辑
func TestTranslateAction(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		key      string
		expected string
	}{
		// 中文翻译
		{"zh_create_users", "zh", ActionCreateUsers, "创建用户"},
		{"zh_delete_roles", "zh", ActionDeleteRoles, "删除角色"},
		{"zh_login", "zh", ActionLogin, "用户登录"},
		{"zh_password_reset", "zh", ActionPasswordReset, "重置密码"},
		{"zh_change_password", "zh", ActionChangePassword, "修改密码"},
		{"zh_update_profile", "zh", ActionUpdateProfile, "更新个人资料"},
		{"zh_upload_avatar", "zh", ActionUploadAvatar, "上传头像"},
		{"zh_cancel_tasks", "zh", ActionCancelTasks, "取消任务"},
		{"zh_view_brand_config", "zh", ActionViewBrandConfig, "查看品牌配置"},
		{"zh_create_feedback", "zh", ActionCreateFeedback, "提交反馈"},
		{"zh_upload_brand_logo", "zh", ActionUploadBrandLogo, "上传品牌Logo"},

		// 英文翻译
		{"en_create_users", "en", ActionCreateUsers, "Create user"},
		{"en_login", "en", ActionLogin, "User login"},
		{"en_password_reset", "en", ActionPasswordReset, "Password reset"},
		{"en_cancel_tasks", "en", ActionCancelTasks, "Cancel task"},
		{"en_create_feedback", "en", ActionCreateFeedback, "Submit feedback"},

		// 繁体中文翻译
		{"zh-tw_create_users", "zh-tw", ActionCreateUsers, "建立使用者"},
		{"zh-tw_password_reset", "zh-tw", ActionPasswordReset, "重設密碼"},
		{"zh-tw_cancel_tasks", "zh-tw", ActionCancelTasks, "取消任務"},

		// 未知 key 回退：返回 key 原文
		{"unknown_key_fallback", "zh", "action.nonexistent", "action.nonexistent"},

		// 未知语言回退到英文
		{"unknown_lang_fallback", "fr", ActionCreateUsers, "Create user"},

		// 空语言回退到英文
		{"empty_lang_fallback", "", ActionCreateUsers, "Create user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslateAction(tt.lang, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTranslateResult 验证结果摘要的翻译逻辑
func TestTranslateResult(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		result   string
		expected string
	}{
		{"zh_success", "zh", "success", "成功"},
		{"zh_failed", "zh", "failed", "失败"},
		{"en_success", "en", "success", "Success"},
		{"en_failed", "en", "failed", "Failed"},
		{"zh-tw_success", "zh-tw", "success", "成功"},
		{"unknown_result_fallback", "zh", "timeout", "timeout"},
		{"empty_lang_fallback", "", "success", "Success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslateResult(tt.lang, tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTranslateResource 验证资源类型的翻译逻辑
func TestTranslateResource(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		raw      string
		expected string
	}{
		// 中文翻译
		{"zh_users", "zh", "users", "用户"},
		{"zh_roles", "zh", "roles", "角色"},
		{"zh_permissions", "zh", "permissions", "权限"},
		{"zh_files", "zh", "files", "文件"},
		{"zh_tasks", "zh", "tasks", "任务"},
		{"zh_dashboard", "zh", "dashboard", "仪表盘"},
		{"zh_mail_config", "zh", "mail-config", "邮件配置"},
		{"zh_brand_config", "zh", "brand-config", "品牌配置"},
		{"zh_feedback", "zh", "feedback", "反馈"},
		{"zh_audit_logs", "zh", "audit-logs", "审计日志"},

		// 英文翻译
		{"en_users", "en", "users", "Users"},
		{"en_brand_config", "en", "brand-config", "Brand Config"},

		// 未知资源类型回退：返回原始值
		{"unknown_resource_fallback", "zh", "unknown-resource", "unknown-resource"},

		// 未知语言回退到英文
		{"unknown_lang_fallback", "fr", "users", "Users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslateResource(tt.lang, tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTranslateMethod 验证 HTTP 请求方法的翻译逻辑
func TestTranslateMethod(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		raw      string
		expected string
	}{
		// 中文翻译
		{"zh_get", "zh", "GET", "查询"},
		{"zh_post", "zh", "POST", "创建"},
		{"zh_put", "zh", "PUT", "更新"},
		{"zh_patch", "zh", "PATCH", "局部更新"},
		{"zh_delete", "zh", "DELETE", "删除"},

		// 英文翻译（保持原始 HTTP 方法名）
		{"en_get", "en", "GET", "GET"},
		{"en_post", "en", "POST", "POST"},

		// 未知方法回退：返回原始值
		{"unknown_method_fallback", "zh", "OPTIONS", "OPTIONS"},

		// 未知语言回退到英文
		{"unknown_lang_fallback", "fr", "GET", "GET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslateMethod(tt.lang, tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAllActionsHaveAllLanguages 验证所有语言的 actionStorage 包含相同的 key 集合
func TestAllActionsHaveAllLanguages(t *testing.T) {
	// 以英文为基准，检查所有语言都覆盖了相同的 key
	enKeys := make(map[string]bool)
	for k := range actionStorage["en"] {
		enKeys[k] = true
	}

	for lang, msgs := range actionStorage {
		t.Run("action_coverage_"+lang, func(t *testing.T) {
			for k := range enKeys {
				_, ok := msgs[k]
				assert.True(t, ok, "language %q missing action key %q", lang, k)
			}
		})
	}
}

// TestAllResourceTranslationsHaveAllLanguages 验证所有语言的 resourceStorage 包含相同的 key 集合
func TestAllResourceTranslationsHaveAllLanguages(t *testing.T) {
	enKeys := make(map[string]bool)
	for k := range resourceStorage["en"] {
		enKeys[k] = true
	}

	for lang, msgs := range resourceStorage {
		t.Run("resource_coverage_"+lang, func(t *testing.T) {
			for k := range enKeys {
				_, ok := msgs[k]
				assert.True(t, ok, "language %q missing resource key %q", lang, k)
			}
		})
	}
}

// TestAllMethodTranslationsHaveAllLanguages 验证所有语言的 methodStorage 包含相同的 key 集合
func TestAllMethodTranslationsHaveAllLanguages(t *testing.T) {
	enKeys := make(map[string]bool)
	for k := range methodStorage["en"] {
		enKeys[k] = true
	}

	for lang, msgs := range methodStorage {
		t.Run("method_coverage_"+lang, func(t *testing.T) {
			for k := range enKeys {
				_, ok := msgs[k]
				assert.True(t, ok, "language %q missing method key %q", lang, k)
			}
		})
	}
}
