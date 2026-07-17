package log

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SafeCore 是一个 zapcore.Core 包装器，在写入日志前自动脱敏包含敏感 key 的字段。
// 敏感 key 列表在构造时固定，匹配时不区分大小写。
type SafeCore struct {
	zapcore.LevelEnabler
	core      zapcore.Core
	keysMap   map[string]bool // precomputed lowercase key map; shared by With() forks
	keys      []string        // retained for With() propagation
}

// NewSafeCore 创建一个新的 SafeCore，包裹给定的 core 并使用指定的敏感 key 列表。
// 当 keys 为空时不做脱敏（空 map），此时 SafeCore 接近透明。
// 敏感 key 列表应通过 config 提供，config.go 的 defaults 是单一事实来源。
func NewSafeCore(core zapcore.Core, keys []string) *SafeCore {
	km := make(map[string]bool, len(keys))
	for _, k := range keys {
		km[strings.ToLower(k)] = true
	}
	return &SafeCore{
		LevelEnabler: core,
		core:         core,
		keys:         keys,
		keysMap:      km,
	}
}

// Check 实现 zapcore.Core 接口。当级别启用时，将自身加入 CheckedEntry 的 cores 列表，
// 使得 Write 调用经过 SafeCore.Write 进行脱敏处理。
func (sc *SafeCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if sc.LevelEnabler.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, sc)
	}
	return checkedEntry
}

// Write 实现 zapcore.Core 接口，在写入前脱敏敏感字段。
func (sc *SafeCore) Write(entry zapcore.Entry, fields []zap.Field) error {
	sanitized := sc.sanitizeFields(fields)
	return sc.core.Write(entry, sanitized)
}

// With 实现 zapcore.Core 接口。
func (sc *SafeCore) With(fields []zap.Field) zapcore.Core {
	wrappedCore := sc.core.With(fields)
	return &SafeCore{
		LevelEnabler: wrappedCore,
		core:         wrappedCore,
		keys:         sc.keys,
		keysMap:      sc.keysMap, // shared by reference, read-only after construction
	}
}

// Sync 实现 zapcore.Core 接口，委托给内部 Core。
func (sc *SafeCore) Sync() error {
	return sc.core.Sync()
}

// sanitizeFields 遍历字段列表，将 key 匹配敏感列表的字段值替换为 "****"。
// 匹配时不区分大小写。使用预计算的 keysMap，避免每次 Write 分配。
func (sc *SafeCore) sanitizeFields(fields []zap.Field) []zap.Field {
	result := make([]zap.Field, len(fields))
	for i, f := range fields {
		if sc.keysMap[strings.ToLower(f.Key)] {
			result[i] = zap.Field{
				Key:    f.Key,
				Type:   zapcore.StringType,
				String: "****",
			}
		} else {
			result[i] = f
		}
	}
	return result
}
