package service

import (
	"errors"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// mapTimeRangeError 将 scopes 包中定义的时间范围哨兵错误映射为统一的业务层应用错误码。
// 例如 scopes.ErrStartTimeFormat 映射为 apperrors.ErrStartTimeFormat (整型码，如 10008)。
// 哨兵错误的详细定义请参见 internal/pkg/scopes/scopes.go。
func mapTimeRangeError(err error) *apperrors.AppError {
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
