// Package scopes 提供 GORM 查询范围辅助函数，包含分页、排序、字段过滤和时间范围筛选。
package scopes

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var validFieldName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Scope 定义了 GORM 查询范围函数的类型签名。
type Scope = func(*gorm.DB) *gorm.DB

// Paginate 返回一个分页查询范围，根据页码和每页数量计算 offset 和 limit。
func Paginate(page, pageSize int) Scope {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// OrderBy returns a Scope that applies ORDER BY with field whitelist validation.
// If sort is empty or not in allowedFields, no ordering is applied.
func OrderBy(sort, order string, allowedFields ...string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if sort == "" || !validFieldName.MatchString(sort) {
			return db
		}
		for _, f := range allowedFields {
			if sort == f {
				dir := "ASC"
				if strings.ToLower(order) == "desc" {
					dir = "DESC"
				}
				return db.Order(sort + " " + dir)
			}
		}
		return db
	}
}

// Eq 返回一个等值查询范围，支持 nil 值（生成 IS NULL 条件）。
func Eq(field string, value interface{}) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if !validFieldName.MatchString(field) {
			zap.L().Warn("scopes.Eq: invalid field name, skipping", zap.String("field", field))
			return db
		}
		if value == nil {
			return db.Where(field + " IS NULL")
		}
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if v.IsNil() {
				return db.Where(field + " IS NULL")
			}
		}
		return db.Where(field+" = ?", value)
	}
}

// Like 返回一个 LIKE 模糊查询范围，在值前后自动添加通配符 %。
func Like(field, value string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if !validFieldName.MatchString(field) {
			zap.L().Warn("scopes.Like: invalid field name, skipping", zap.String("field", field))
			return db
		}
		return db.Where(field+" LIKE ?", "%"+value+"%")
	}
}

// MultiLike generates an OR-combined LIKE condition across multiple fields.
// Each field name is validated to prevent SQL injection.
func MultiLike(fields []string, value string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		conds := make([]string, 0, len(fields))
		args := make([]interface{}, 0, len(fields))
		for _, f := range fields {
			if !validFieldName.MatchString(f) {
				continue
			}
			conds = append(conds, f+" LIKE ?")
			args = append(args, "%"+value+"%")
		}
		if len(conds) == 0 {
			return db
		}
		return db.Where(strings.Join(conds, " OR "), args...)
	}
}

// TimeRange 返回一个时间范围查询范围，支持开始时间和结束时间单独或同时指定。
func TimeRange(field string, from, to *time.Time) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if !validFieldName.MatchString(field) {
			zap.L().Warn("scopes.TimeRange: invalid field name, skipping", zap.String("field", field))
			return db
		}
		if from != nil {
			db = db.Where(field+" >= ?", *from)
		}
		if to != nil {
			db = db.Where(field+" <= ?", *to)
		}
		return db
	}
}

// Sentinel errors for ParseTimeRange validation.
// These are internal sentinel errors used for errors.Is matching in time_helpers.go,
// which maps them to apperrors error codes (ErrStartTimeFormat=10008, etc.).
// Naming mirrors apperrors constants intentionally; see internal/pkg/errors/errors.go.
var (
	ErrStartTimeFormat = errors.New("start time format invalid")
	ErrEndTimeFormat   = errors.New("end time format invalid")
	ErrTimeRangeOrder  = errors.New("start time must not be after end time")
)

var supportedTimeFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// ParseTimeRange 解析时间范围字符串为 *time.Time 值
//
// 【核心功能】将前端传递的 start_time/end_time 字符串解析为 *time.Time，
// 支持多种时间格式（RFC3339、DateTime、Date），用于 Audit/File/Task 等资源的时间范围筛选。
//
// 注意：非 RFC3339 格式（如 "2024-01-15"、"2024-01-15 10:30:00"）解析结果为 UTC 时区。
// 数据库 created_at 字段应统一使用 UTC 存储，否则需在调用方做时区转换。
//
// 参数:
//   - startStr: 开始时间字符串，可为空
//   - endStr: 结束时间字符串，可为空
//
// 返回值:
//   - from: 开始时间，未传入时为 nil
//   - to: 结束时间，未传入时为 nil
//   - error: 时间格式错误或开始时间晚于结束时间时返回
func ParseTimeRange(startStr, endStr string) (from, to *time.Time, err error) {
	if startStr != "" {
		parsed, e := parseTimeWithFormats(startStr, supportedTimeFormats)
		if e != nil {
			return nil, nil, ErrStartTimeFormat
		}
		from = &parsed
	}

	if endStr != "" {
		parsed, e := parseTimeWithFormats(endStr, supportedTimeFormats)
		if e != nil {
			return nil, nil, ErrEndTimeFormat
		}
		if isDateOnly(endStr) {
			parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 999999999, parsed.Location())
		}
		to = &parsed
	}

	if from != nil && to != nil && from.After(*to) {
		return nil, nil, ErrTimeRangeOrder
	}

	return from, to, nil
}

// isDateOnly detects whether a time string represents a date without clock time.
// Uses time.Parse to verify the string matches "2006-01-02" exactly, which also
// rejects invalid dates like 2024-02-30. This function is only called after
// parseTimeWithFormats has already succeeded, so a parse failure here indicates
// the string contains clock-time components rather than an invalid date.
func isDateOnly(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// parseTimeWithFormats attempts to parse a time string using multiple formats.
func parseTimeWithFormats(value string, formats []string) (time.Time, error) {
	var lastErr error
	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}
