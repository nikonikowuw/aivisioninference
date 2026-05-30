// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

const (
	defaultSystemName = "Niko Admin"
	defaultLogoURL    = "/favicon.ico"
	maxBrandLogoSize  = 2 * 1024 * 1024
	brandLogoPrefix   = "brand"
)

// brandConfigRepo 品牌配置持久化接口。
type brandConfigRepo interface {
	First(ctx context.Context) (*model.BrandConfig, error)
	Save(ctx context.Context, cfg *model.BrandConfig) error
}

// BrandService 处理系统品牌配置的业务逻辑。
type BrandService struct {
	cfgRepo brandConfigRepo
	storage storage.Storage
}

// NewBrandService 创建并返回一个新的 BrandService 实例。
func NewBrandService(cfgRepo brandConfigRepo) *BrandService {
	return &BrandService{cfgRepo: cfgRepo}
}

// NewBrandServiceWithStorage 创建带 Logo 存储能力的 BrandService 实例。
func NewBrandServiceWithStorage(cfgRepo brandConfigRepo, stor storage.Storage) *BrandService {
	return &BrandService{cfgRepo: cfgRepo, storage: stor}
}

// GetConfig 获取当前品牌配置，不存在时返回默认品牌配置。
func (s *BrandService) GetConfig(ctx context.Context) (*dto.BrandConfigResponse, error) {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		zap.L().Error("get brand config failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return toBrandConfigResponse(cfg), nil
}

// SaveConfig 保存或更新系统品牌配置。
func (s *BrandService) SaveConfig(ctx context.Context, req dto.BrandConfigRequest) (*dto.BrandConfigResponse, error) {
	cfg, err := s.getOrDefaultConfig(ctx)
	if err != nil {
		zap.L().Error("get brand config before save failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	oldLogoURL := cfg.LogoURL
	cfg.SystemName = strings.TrimSpace(req.SystemName)
	cfg.LogoURL = strings.TrimSpace(req.LogoURL)
	if err := s.cfgRepo.Save(ctx, cfg); err != nil {
		zap.L().Error("save brand config failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	if oldLogoURL != cfg.LogoURL {
		s.deleteOldLogo(oldLogoURL)
	}

	return toBrandConfigResponse(cfg), nil
}

// UploadLogo 校验并上传系统 Logo，成功后同步更新品牌配置。
func (s *BrandService) UploadLogo(ctx context.Context, fileHeader *multipart.FileHeader) (string, error) {
	if s.storage == nil {
		return "", apperrors.New(apperrors.ErrInternal, "")
	}
	if fileHeader.Size > maxBrandLogoSize {
		return "", apperrors.New(apperrors.ErrFileTooLarge, "")
	}

	allBytes, err := readUploadedLogo(fileHeader)
	if err != nil {
		return "", err
	}
	mimeType := http.DetectContentType(allBytes[:512])
	if !allowedAvatarTypes[mimeType] {
		return "", apperrors.New(apperrors.ErrFileInvalidType, "")
	}

	storagePath := brandLogoPrefix + "/" + uuid.New().String() + logoExtension(fileHeader.Filename, mimeType)
	storedPath, err := s.storage.Save(bytes.NewReader(allBytes), storagePath)
	if err != nil {
		zap.L().Error("save brand logo failed", zap.Error(err))
		return "", apperrors.New(apperrors.ErrInternal, "")
	}

	logoURL := s.storage.GetURL(storedPath)
	return logoURL, nil
}

// getOrDefaultConfig 获取品牌配置，不存在时返回包含默认值的占位配置。
func (s *BrandService) getOrDefaultConfig(ctx context.Context) (*model.BrandConfig, error) {
	cfg, err := s.cfgRepo.First(ctx)
	if err != nil {
		return nil, err
	}
	if cfg != nil {
		return cfg, nil
	}
	return &model.BrandConfig{SystemName: defaultSystemName, LogoURL: defaultLogoURL}, nil
}

// toBrandConfigResponse 将 BrandConfig 模型转换为 DTO 响应结构体。
func toBrandConfigResponse(cfg *model.BrandConfig) *dto.BrandConfigResponse {
	return &dto.BrandConfigResponse{
		ID:         cfg.ID,
		SystemName: cfg.SystemName,
		LogoURL:    cfg.LogoURL,
		CreatedAt:  cfg.CreatedAt.Format(dto.DateTimeFormat),
		UpdatedAt:  cfg.UpdatedAt.Format(dto.DateTimeFormat),
	}
}

// readUploadedLogo 读取并校验上传的品牌 Logo 文件内容（大小和完整性校验）。
func readUploadedLogo(fileHeader *multipart.FileHeader) ([]byte, error) {
	file, err := fileHeader.Open()
	if err != nil {
		zap.L().Error("open uploaded brand logo failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	defer file.Close()

	allBytes, err := io.ReadAll(file)
	if err != nil {
		zap.L().Error("read uploaded brand logo failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if len(allBytes) > maxBrandLogoSize {
		return nil, apperrors.New(apperrors.ErrFileTooLarge, "")
	}
	if len(allBytes) < 512 {
		return nil, apperrors.New(apperrors.ErrBadRequest, "文件内容不完整")
	}
	return allBytes, nil
}

// logoExtension 根据文件名和 MIME 类型推断 Logo 文件的扩展名。
func logoExtension(filename, mimeType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".jpeg" {
		return ".jpg"
	}
	if ext != "" {
		return ext
	}
	if mimeType == "image/jpeg" {
		return ".jpg"
	}
	exts, _ := mime.ExtensionsByType(mimeType)
	if len(exts) > 0 {
		return exts[0]
	}
	return ""
}

// deleteOldLogo 尽力删除旧 Logo 文件，仅删除本存储后端的文件。
func (s *BrandService) deleteOldLogo(oldLogoURL string) {
	if oldLogoURL == "" || oldLogoURL == defaultLogoURL {
		return
	}
	baseURL := s.storage.GetURL("")
	if !strings.HasPrefix(oldLogoURL, baseURL) {
		return
	}
	oldPath := strings.TrimPrefix(oldLogoURL, baseURL+"/")
	if err := s.storage.Delete(oldPath); err != nil {
		zap.L().Warn("delete old brand logo failed", zap.String("path", oldPath), zap.Error(err))
	}
}
