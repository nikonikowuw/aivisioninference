// Package timex 提供时间范围解析辅助函数。
package timex

import (
	"errors"
	"time"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// MapRangeError 将 scopes 包中定义的时间范围哨兵错误映射为统一的业务层应用错误码。
func MapRangeError(err error) *apperrors.AppError {
	switch {
	case errors.Is(err, scopes.ErrStartTimeFormat):
		return apperrors.New(apperrors.ErrStartTimeFormat, "")
	case errors.Is(err, scopes.ErrEndTimeFormat):
		return apperrors.New(apperrors.ErrEndTimeFormat, "")
	case errors.Is(err, scopes.ErrTimeRangeOrder):
		return apperrors.New(apperrors.ErrTimeRangeOrder, "")
	default:
		return apperrors.New(apperrors.ErrBadRequest, "")
	}
}

// NormalizeRange 校验并解析开始/结束时间字符串，将结果写入 fromTime/toTime 指针。
func NormalizeRange(startTime, endTime string, fromTime **time.Time, toTime **time.Time) error {
	if startTime == "" && endTime == "" {
		return nil
	}
	from, to, err := scopes.ParseTimeRange(startTime, endTime)
	if err != nil {
		return MapRangeError(err)
	}
	*fromTime = from
	*toTime = to
	return nil
}
