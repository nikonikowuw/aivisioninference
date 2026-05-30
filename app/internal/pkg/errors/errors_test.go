package errors

import "testing"

func TestNewMarksDefaultMessage(t *testing.T) {
	err := New(ErrUnauthorized, "")
	if !err.IsDefaultMessage() {
		t.Fatal("expected empty message to be marked as default")
	}
	if err.Message != DefaultMessage(ErrUnauthorized, "en") {
		t.Fatalf("expected en default message, got %q", err.Message)
	}
}

func TestNewKeepsExplicitMessage(t *testing.T) {
	err := New(ErrUnauthorized, "缺少认证令牌")
	if err.IsDefaultMessage() {
		t.Fatal("expected explicit message not to be marked as default")
	}
	if err.Message != "缺少认证令牌" {
		t.Fatalf("expected explicit message, got %q", err.Message)
	}
}

func TestDefaultMessageFallback(t *testing.T) {
	if got := DefaultMessage(ErrUnauthorized, "unsupported"); got != DefaultMessage(ErrUnauthorized, "en") {
		t.Fatalf("expected unsupported language fallback to en, got %q", got)
	}
	if got := DefaultMessage(99999, "zh"); got != "Unknown error" {
		t.Fatalf("expected unknown code fallback, got %q", got)
	}
}

func TestDefaultMessageSupportsMiddlewareLanguages(t *testing.T) {
	tests := map[string]string{
		"zh":    "未登录",
		"zh-tw": "未登入",
		"en":    "Unauthorized",
		"id":    "Tidak sah",
		"ja":    "認証されていません",
		"ko":    "인증되지 않았습니다",
	}

	for lang, want := range tests {
		if got := DefaultMessage(ErrUnauthorized, lang); got != want {
			t.Fatalf("expected %s message %q, got %q", lang, want, got)
		}
	}
}
