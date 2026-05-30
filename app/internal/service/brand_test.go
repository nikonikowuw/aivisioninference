package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
)

type fakeBrandConfigRepo struct {
	cfg *model.BrandConfig
	err error
}

func (f *fakeBrandConfigRepo) First(_ context.Context) (*model.BrandConfig, error) {
	return f.cfg, f.err
}
func (f *fakeBrandConfigRepo) Save(_ context.Context, cfg *model.BrandConfig) error {
	f.cfg = cfg
	return f.err
}

func TestBrandServiceGetConfigUsesDefaults(t *testing.T) {
	svc := NewBrandService(&fakeBrandConfigRepo{})

	res, err := svc.GetConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, defaultSystemName, res.SystemName)
	require.Equal(t, defaultLogoURL, res.LogoURL)
}

func TestBrandServiceSaveConfigTrimsAndDefaults(t *testing.T) {
	repo := &fakeBrandConfigRepo{}
	svc := NewBrandService(repo)

	res, err := svc.SaveConfig(context.Background(), dto.BrandConfigRequest{
		SystemName: "  Custom Admin  ",
		LogoURL:    "  /uploads/brand/logo.png  ",
	})

	require.NoError(t, err)
	require.Equal(t, "Custom Admin", res.SystemName)
	require.Equal(t, "/uploads/brand/logo.png", res.LogoURL)
	require.Equal(t, "Custom Admin", repo.cfg.SystemName)
	require.Equal(t, "/uploads/brand/logo.png", repo.cfg.LogoURL)
}
