// Package service 提供业务逻辑层实现。
package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/fingerprint"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// LicenseClaims JWT Payload 结构
type LicenseClaims struct {
	LicenseID         string   `json:"license_id"`
	LicenseType       string   `json:"license_type"`
	DeviceSN          string   `json:"device_sn,omitempty"`
	DeviceFingerprint string   `json:"device_fingerprint,omitempty"`
	Algorithms        []string `json:"algorithms"`
	MaxStreams        int      `json:"max_streams,omitempty"`
	Features          []string `json:"features,omitempty"`
	ExpireAction      string   `json:"expire_action,omitempty"`
	jwt.RegisteredClaims
}

// LicenseService 授权管理业务逻辑
type LicenseService struct {
	repo         *repository.LicenseRepository
	db           *gorm.DB
	rsaPublicKey *rsa.PublicKey
}

// NewLicenseService 创建授权服务实例
func NewLicenseService(repo *repository.LicenseRepository, db *gorm.DB) *LicenseService {
	svc := &LicenseService{
		repo: repo,
		db:   db,
	}
	// 延迟加载公钥：首次使用时加载，避免构造函数副作用
	return svc
}

// ensurePublicKey 确保 RSA 公钥已加载（延迟初始化）
func (s *LicenseService) ensurePublicKey() error {
	if s.rsaPublicKey != nil {
		return nil
	}
	s.loadPublicKey()
	if s.rsaPublicKey == nil {
		return errors.New(errors.ErrLicenseNotConfigured, "")
	}
	return nil
}

// loadPublicKey 从环境变量或配置文件加载 RSA 公钥
func (s *LicenseService) loadPublicKey() {
	pubKeyPEM := os.Getenv("NIKO_LICENSE_PUBLIC_KEY")
	if pubKeyPEM == "" {
		data, err := os.ReadFile("configs/license_public_key.pem")
		if err != nil {
			zap.L().Warn("license RSA public key not configured, license verification will be unavailable",
				zap.String("env", "NIKO_LICENSE_PUBLIC_KEY"),
				zap.String("file", "configs/license_public_key.pem"))
			return
		}
		pubKeyPEM = string(data)
	}

	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		zap.L().Error("failed to decode license RSA public key PEM")
		return
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		zap.L().Error("failed to parse license RSA public key", zap.Error(err))
		return
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		zap.L().Error("license public key is not RSA type")
		return
	}

	s.rsaPublicKey = rsaPub
	zap.L().Info("license RSA public key loaded successfully")
}

// GetDeviceFingerprint 返回本机硬件指纹（供前端展示）
func (s *LicenseService) GetDeviceFingerprint(ctx context.Context) (*dto.FingerprintResponse, error) {
	deviceSN := s.getDeviceSN(ctx)

	fp, err := fingerprint.Extract()
	if err != nil {
		zap.L().Error("failed to extract device fingerprint", zap.Error(err))
		return nil, errors.New(errors.ErrFingerprintExtract, "")
	}

	return &dto.FingerprintResponse{
		DeviceSN:      deviceSN,
		Fingerprint:   fp,
		HashAlgorithm: "SHA-256",
	}, nil
}

// Upload 上传并解析 .lic 授权文件
func (s *LicenseService) Upload(ctx context.Context, rawJWT string) (*dto.LicenseInfoResponse, error) {
	if err := s.ensurePublicKey(); err != nil {
		return nil, err
	}

	// 解析 JWT
	claims := &LicenseClaims{}
	token, err := jwt.ParseWithClaims(rawJWT, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.rsaPublicKey, nil
	})
	if err != nil {
		zap.L().Warn("license JWT parsing failed", zap.Error(err))
		return nil, errors.New(errors.ErrLicenseInvalid, "")
	}
	if !token.Valid {
		return nil, errors.New(errors.ErrLicenseInvalid, "")
	}

	// 校验设备指纹匹配
	localFingerprint, fpErr := fingerprint.Extract()
	if fpErr != nil {
		zap.L().Warn("failed to extract local fingerprint for license verification", zap.Error(fpErr))
	}
	if claims.DeviceFingerprint != "" && localFingerprint != "" {
		if claims.DeviceFingerprint != localFingerprint {
			return nil, errors.New(errors.ErrLicenseDeviceMismatch, "")
		}
	}

	// 计算授权状态
	now := time.Now()
	status := model.LicenseStatusActive
	if now.Before(claims.NotBefore.Time) {
		status = model.LicenseStatusNotYetValid
	} else if claims.ExpiresAt != nil && now.After(claims.ExpiresAt.Time) {
		status = model.LicenseStatusExpired
	}

	// 默认过期动作
	expireAction := claims.ExpireAction
	if expireAction == "" {
		expireAction = model.ExpireActionKeepRunning
	}

	// 序列化 JSON 字段
	algorithmsJSON, _ := json.Marshal(claims.Algorithms)
	featuresJSON, _ := json.Marshal(claims.Features)

	// 构造模型
	license := &model.DeviceLicense{
		LicenseID:         claims.LicenseID,
		LicenseType:       claims.LicenseType,
		DeviceSN:          claims.DeviceSN,
		DeviceFingerprint: claims.DeviceFingerprint,
		Algorithms:        algorithmsJSON,
		MaxStreams:        claims.MaxStreams,
		Features:          featuresJSON,
		NotBefore:         claims.NotBefore.Time,
		NotAfter:          getExpirationTime(claims.ExpiresAt),
		ExpireAction:      expireAction,
		Status:            status,
		Signature:         rawJWT,
		RawContent:        rawJWT,
	}

	// 检查是否已存在相同 LicenseID 的授权
	existing, findErr := s.repo.FindByLicenseID(ctx, claims.LicenseID)
	if findErr == nil && existing != nil {
		license.ID = existing.ID
		license.CreatedBy = existing.CreatedBy
		license.CreatedAt = existing.CreatedAt
		if err := s.repo.Update(ctx, license); err != nil {
			zap.L().Error("update license failed", zap.Error(err))
			return nil, errors.New(errors.ErrInternal, "")
		}
	} else {
		if err := s.repo.Create(ctx, license); err != nil {
			zap.L().Error("create license failed", zap.Error(err))
			return nil, errors.New(errors.ErrInternal, "")
		}
	}

	// 审计
	zap.L().Info("license imported",
		zap.String("license_id", license.LicenseID),
		zap.String("license_type", license.LicenseType),
		zap.String("status", status))

	return s.toResponse(license), nil
}

// GetActive 获取当前生效的授权信息
func (s *LicenseService) GetActive(ctx context.Context) (*dto.LicenseInfoResponse, error) {
	license, err := s.repo.FindActive(ctx)
	if err != nil {
		return nil, errors.New(errors.ErrLicenseNotFound, "")
	}
	return s.toResponse(license), nil
}

// List 分页查询授权列表
func (s *LicenseService) List(ctx context.Context, req dto.LicenseListRequest) ([]dto.LicenseInfoResponse, int64, error) {
	items, total, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	var results []dto.LicenseInfoResponse
	for _, item := range items {
		results = append(results, *s.toResponse(&item))
	}
	return results, total, nil
}

// CheckAlgorithmAuth 校验指定算法是否被授权且授权有效
func (s *LicenseService) CheckAlgorithmAuth(ctx context.Context, algoName string) error {
	license, err := s.repo.FindActive(ctx)
	if err != nil {
		return errors.New(errors.ErrLicenseNotAuthorized, "")
	}

	// 先检查授权是否生效，再检查算法是否在授权范围内
	if !license.IsEffective() {
		if license.IsExpired() {
			return errors.New(errors.ErrLicenseExpired, "")
		}
		return errors.New(errors.ErrLicenseNotYetValid, "")
	}

	if !license.HasAlgorithm(algoName) {
		return errors.New(errors.ErrLicenseNotAuthorized, "")
	}

	return nil
}

// CheckStreamLimit 校验当前运行任务数是否超过授权路数限制
func (s *LicenseService) CheckStreamLimit(ctx context.Context, currentRunningStreams int) error {
	license, err := s.repo.FindActive(ctx)
	if err != nil {
		return errors.New(errors.ErrLicenseNotAuthorized, "")
	}

	if license.MaxStreams > 0 && currentRunningStreams >= license.MaxStreams {
		return errors.New(errors.ErrLicenseStreamLimit, "")
	}

	return nil
}

// RefreshExpiredStatus 巡检并刷新过期授权状态（由后台定时任务调用）
func (s *LicenseService) RefreshExpiredStatus(ctx context.Context) []string {
	var expiredLicenseIDs []string
	now := time.Now()

	req := dto.LicenseListRequest{Status: model.LicenseStatusActive}
	items, _, err := s.repo.List(ctx, req)
	if err != nil {
		zap.L().Error("failed to list active licenses for expiration check", zap.Error(err))
		return nil
	}

	for _, license := range items {
		if license.NotAfter != nil && now.After(*license.NotAfter) {
			if err := s.repo.UpdateStatus(ctx, license.ID, model.LicenseStatusExpired); err != nil {
				zap.L().Error("failed to update expired license status",
					zap.String("license_id", license.LicenseID), zap.Error(err))
				continue
			}
			expiredLicenseIDs = append(expiredLicenseIDs, license.LicenseID)
			zap.L().Info("license expired by patrol",
				zap.String("license_id", license.LicenseID),
				zap.Time("not_after", *license.NotAfter))
		}
	}

	return expiredLicenseIDs
}

// toResponse 将模型转换为响应 DTO
func (s *LicenseService) toResponse(license *model.DeviceLicense) *dto.LicenseInfoResponse {
	var algorithms []string
	if len(license.Algorithms) > 0 {
		if err := json.Unmarshal(license.Algorithms, &algorithms); err != nil {
			zap.L().Warn("failed to unmarshal license algorithms",
				zap.String("license_id", license.LicenseID), zap.Error(err))
		}
	}
	var features []string
	if len(license.Features) > 0 {
		if err := json.Unmarshal(license.Features, &features); err != nil {
			zap.L().Warn("failed to unmarshal license features",
				zap.String("license_id", license.LicenseID), zap.Error(err))
		}
	}

	var remainingDays *int
	if license.NotAfter != nil {
		days := int(time.Until(*license.NotAfter).Hours() / 24)
		if days < 0 {
			days = 0
		}
		remainingDays = &days
	}

	return &dto.LicenseInfoResponse{
		ID:                license.ID,
		LicenseID:         license.LicenseID,
		LicenseType:       license.LicenseType,
		DeviceSN:          license.DeviceSN,
		DeviceFingerprint: license.DeviceFingerprint,
		Algorithms:        algorithms,
		MaxStreams:        license.MaxStreams,
		Features:          features,
		NotBefore:         license.NotBefore,
		NotAfter:          license.NotAfter,
		ExpireAction:      license.ExpireAction,
		Status:            license.Status,
		RemainingDays:     remainingDays,
		CreatedAt:         license.CreatedAt,
		UpdatedAt:         license.UpdatedAt,
	}
}

// getDeviceSN 从数据库读取设备 SN
func (s *LicenseService) getDeviceSN(ctx context.Context) string {
	var config model.SystemConfig
	if err := s.db.WithContext(ctx).Where("id = ?", "default").First(&config).Error; err == nil {
		return config.DeviceSN
	}
	return ""
}

// getExpirationTime 从 jwt.NumericDate 提取 time.Time（允许永久授权为空）
func getExpirationTime(nd *jwt.NumericDate) *time.Time {
	if nd == nil {
		return nil
	}
	t := nd.Time
	return &t
}
