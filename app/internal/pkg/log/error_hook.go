package log

import (
	"bytes"
	"encoding/json"
	"net/http"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ErrorEntry 封装一条触发告警的日志条目信息。
type ErrorEntry struct {
	Time    time.Time    `json:"time"`
	Message string       `json:"message"`
	Level   string       `json:"level"`
	Fields  []zap.Field  `json:"-"`
	Stack   string       `json:"stack,omitempty"`
	Caller  string       `json:"caller,omitempty"`
}

// ErrorHook 是错误级别日志的回调接口，实现方通过 OnError 接收告警事件。
type ErrorHook interface {
	OnError(entry ErrorEntry)
}

// ErrorHookFunc 方便将普通函数适配为 ErrorHook 接口。
type ErrorHookFunc func(entry ErrorEntry)

// OnError 实现 ErrorHook 接口。
func (f ErrorHookFunc) OnError(entry ErrorEntry) {
	f(entry)
}

// WebhookErrorHook 将错误日志事件通过 HTTP POST JSON 发送到配置的 URL。
type WebhookErrorHook struct {
	url    string
	client *http.Client
}

// NewWebhookErrorHook 创建一个向指定 URL POST 错误事件的 WebhookErrorHook。
func NewWebhookErrorHook(url string) *WebhookErrorHook {
	return &WebhookErrorHook{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// OnError 实现 ErrorHook 接口，将错误事件异步发送到 webhook URL。
func (h *WebhookErrorHook) OnError(entry ErrorEntry) {
	go func() {
		body, err := json.Marshal(entry)
		if err != nil {
			return
		}
		_, _ = h.client.Post(h.url, "application/json", bytes.NewReader(body))
	}()
}

// HookCore 是一个 zapcore.Core 包装器，在 Write 时检查日志级别，
// 对达到 minimumLevel 的日志条目异步调用注册的 ErrorHook 回调。
type HookCore struct {
	zapcore.LevelEnabler
	core         zapcore.Core
	hooks        []ErrorHook
	minimumLevel zapcore.Level
}

// NewHookCore 创建一个 HookCore，包裹给定的 core。
// 当写入的日志级别 >= minimumLevel 时，同步触发所有 hooks。
func NewHookCore(core zapcore.Core, minimumLevel zapcore.Level, hooks []ErrorHook) *HookCore {
	if len(hooks) == 0 {
		hooks = nil
	}
	return &HookCore{
		LevelEnabler: core,
		core:         core,
		hooks:        hooks,
		minimumLevel: minimumLevel,
	}
}

// Check 实现 zapcore.Core 接口。当级别启用时，将自身加入 CheckedEntry 的 cores 列表。
func (hc *HookCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if hc.LevelEnabler.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, hc)
	}
	return checkedEntry
}

// Write 实现 zapcore.Core 接口，在写入前检查级别并触发 hooks。
func (hc *HookCore) Write(entry zapcore.Entry, fields []zap.Field) error {
	if entry.Level >= hc.minimumLevel && len(hc.hooks) > 0 {
		// Capture stack and caller
		caller := entry.Caller.TrimmedPath()
		stack := captureStack(3)

		hookEntry := ErrorEntry{
			Time:    entry.Time,
			Message: entry.Message,
			Level:   entry.Level.String(),
			Fields:  fields,
			Stack:   stack,
			Caller:  caller,
		}

		for _, hook := range hc.hooks {
			hook.OnError(hookEntry)
		}
	}
	return hc.core.Write(entry, fields)
}

// With 实现 zapcore.Core 接口。
func (hc *HookCore) With(fields []zap.Field) zapcore.Core {
	return &HookCore{
		LevelEnabler: hc.core,
		core:         hc.core.With(fields),
		hooks:        hc.hooks,
		minimumLevel: hc.minimumLevel,
	}
}

// Sync 实现 zapcore.Core 接口。
func (hc *HookCore) Sync() error {
	return hc.core.Sync()
}

// captureStack 捕获调用栈的前若干帧，返回格式化的栈字符串。
func captureStack(skip int) string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	pcs = pcs[:n]

	var buf strings.Builder
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		// Skip internal runtime and hook core frames
		if strings.Contains(frame.Function, "runtime.") {
			continue
		}
		buf.WriteString(frame.Function)
		buf.WriteString("\n\t")
		buf.WriteString(frame.File)
		buf.WriteString(":")
		buf.WriteString(itoa(frame.Line))
		buf.WriteString("\n")
	}
	return buf.String()
}

// itoa 快速将 int 转为字符串。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
