// Package i18n provides internationalization support for the niko-admin application.
// It maps business error codes to translated messages in multiple languages.
package i18n

import (
	"fmt"
	"sync"
)

const DefaultLanguage = "en"

var (
	mu      sync.RWMutex
	storage = map[string]map[int]string{
		"en": {
			0:     "success",
			10001: "bad request",
			10002: "cannot disable yourself",
			10003: "no permission to operate on same or higher level user",
			10004: "no permission to operate on same or higher level role, or set a role level higher than your own",
			10005: "email already taken",
			10006: "old password is incorrect",
			20001: "unauthorized",
			20002: "token expired",
			20003: "invalid token",
			20403: "refresh token reuse detected",
			30001: "forbidden",
			40001: "resource not found",
			50001: "internal server error",
		},
		"zh": {
			0:     "成功",
			10001: "请求参数错误",
			10002: "不能禁用自己",
			10003: "没有权限操作同级或更高级别的用户",
			10004: "没有权限操作同级或更高级别的角色，也不能设置高于自己权限的角色等级",
			10005: "邮箱已被使用",
			10006: "旧密码错误",
			20001: "未登录",
			20002: "Token已过期",
			20003: "Token无效",
			20403: "Token已被复用，所有设备已强制登出",
			30001: "无权限",
			40001: "资源不存在",
			50001: "服务器内部错误",
		},
		"zh-tw": {
			0:     "成功",
			10001: "請求參數錯誤",
			10002: "不能停用自己",
			10003: "沒有權限操作同級或更高級別的使用者",
			10004: "沒有權限操作同級或更高級別的角色，也不能設定高於自己權限的角色等級",
			10005: "信箱已被使用",
			10006: "舊密碼錯誤",
			20001: "未登入",
			20002: "Token已過期",
			20003: "Token無效",
			20403: "Token已被重複使用，所有裝置已強制登出",
			30001: "無權限",
			40001: "資源不存在",
			50001: "伺服器內部錯誤",
		},
	}
)

// Translate returns the translated message for the given error code and language.
// If lang is empty, it falls back to "en". If the language or code is not found,
// it falls back to English. If still not found, returns "unknown error".
func Translate(lang string, code int) string {
	if lang == "" {
		lang = DefaultLanguage
	}

	mu.RLock()
	defer mu.RUnlock()

	// Try the requested language first.
	if msgs, ok := storage[lang]; ok {
		if msg, ok := msgs[code]; ok {
			return msg
		}
	}

	// Fallback to English.
	if msgs, ok := storage[DefaultLanguage]; ok {
		if msg, ok := msgs[code]; ok {
			return msg
		}
	}

	return fmt.Sprintf("unknown error (code=%d)", code)
}

// RegisterMessages adds or overwrites translations for the given language.
// It is safe to call concurrently.
func RegisterMessages(lang string, msgs map[int]string) {
	mu.Lock()
	defer mu.Unlock()

	if storage[lang] == nil {
		storage[lang] = make(map[int]string)
	}
	for k, v := range msgs {
		storage[lang][k] = v
	}
}

// Languages returns the list of currently registered languages.
func Languages() []string {
	mu.RLock()
	defer mu.RUnlock()

	langs := make([]string, 0, len(storage))
	for lang := range storage {
		langs = append(langs, lang)
	}
	return langs
}

// HasLanguage reports whether translations exist for the given language code.
func HasLanguage(lang string) bool {
	mu.RLock()
	defer mu.RUnlock()

	_, ok := storage[lang]
	return ok
}
