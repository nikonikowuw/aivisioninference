// Package middleware 提供 Gin HTTP 中间件，包含认证鉴权、RBAC 权限控制、审计日志、CORS、限流等功能。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyLang is the gin context key for the resolved language.
	ContextKeyLang = "lang"
	// DefaultLanguage is the fallback language when Accept-Language is missing.
	DefaultLanguage = "en"
)

// supportedLanguages is the set of languages this application supports.
var supportedLanguages = map[string]bool{
	"en":    true,
	"zh":    true,
	"zh-tw": true,
	"id":    true,
	"ja":    true,
	"ko":    true,
}

// I18n returns a Gin middleware that reads the Accept-Language header and
// sets a "lang" key in the gin.Context. Supported values: "en", "zh", "zh-tw", "id", "ja", "ko".
// Falls back to "en" if the header is missing or contains an unsupported language.
func I18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := parseAcceptLanguage(c.GetHeader("Accept-Language"))
		c.Set(ContextKeyLang, lang)
		c.Next()
	}
}

// parseAcceptLanguage extracts the preferred language from an Accept-Language
// header value. It returns the first supported language found, or the default.
//
// Examples:
//
//	"zh-CN,zh;q=0.9,en;q=0.8" → "zh"
//	"zh-TW,zh-HK;q=0.9,en;q=0.8" → "zh-tw"
//	"en-US,en;q=0.9"           → "en"
//	"ja,ko;q=0.9"              → "en" (fallback)
func parseAcceptLanguage(header string) string {
	if header == "" {
		return DefaultLanguage
	}

	// Split by comma: "zh-CN,zh;q=0.9,en;q=0.8"
	langs := strings.Split(header, ",")
	for _, entry := range langs {
		// Remove quality factor: "zh-CN;q=0.9" → "zh-CN"
		lang := strings.TrimSpace(strings.SplitN(entry, ";", 2)[0])

		// Try matching the full language tag first (e.g. "zh-TW" → "zh-tw")
		full := strings.ToLower(lang)
		if supportedLanguages[full] {
			return full
		}

		// Fall back to base language: "zh-CN" → "zh"
		base := strings.ToLower(strings.SplitN(lang, "-", 2)[0])
		if supportedLanguages[base] {
			return base
		}
	}

	return DefaultLanguage
}
