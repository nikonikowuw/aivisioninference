package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

func TestRoleServiceCreateRejectsMissingLevelWithDefaultBadRequestMessage(t *testing.T) {
	svc := &RoleService{}

	role, err := svc.Create(context.Background(), dto.CreateRoleRequest{Name: "manager"}, "", true)

	require.Nil(t, role)
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.ErrBadRequest, appErr.Code)
	assert.Equal(t, apperrors.New(apperrors.ErrBadRequest, "").Message, appErr.Message)
}
