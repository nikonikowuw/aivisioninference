package service

import (
	"errors"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// mapTimeRangeError maps scopes sentinel errors to apperrors error codes.
// scopes.ErrXxx (sentinel error) → apperrors.ErrXxx (int code, e.g. 10008).
// See internal/pkg/scopes/scopes.go for sentinel definitions.
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
