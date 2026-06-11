package service

import (
	"archive/tar"
	"archive/zip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

// AlgorithmOptions defines options for algorithm package processing
type AlgorithmOptions struct {
	MaxAlgoFileSizeBytes int64
	LocalUploadDir       string
	PublicURL            string
}

// AlgoMeta represents the metadata from algo_meta.yaml
type AlgoMeta struct {
	Algorithm         string   `yaml:"algorithm"`
	AlgorithmName     string   `yaml:"algorithm_name"`
	AlgorithmAlias    string   `yaml:"algorithm_alias"`
	Version           string   `yaml:"version"`
	Domain            string   `yaml:"domain"`
	ResultSchema      string   `yaml:"result_schema"`
	CapabilitiesImage []string `yaml:"capabilities_image"`
	CapabilitiesData  []string `yaml:"capabilities_data"`
	Capabilities      struct {
		Image []string `yaml:"image"`
		Data  []string `yaml:"data"`
	} `yaml:"capabilities"`
	Hardware       []string    `yaml:"hardware"`
	Description    string      `yaml:"description"`
	AIParamsSchema interface{} `yaml:"ai_params_schema"`
}

func (m *AlgoMeta) Normalize() {
	if m.AlgorithmName == "" {
		m.AlgorithmName = m.Algorithm
	}
	if len(m.CapabilitiesImage) == 0 {
		m.CapabilitiesImage = m.Capabilities.Image
	}
	if len(m.CapabilitiesData) == 0 {
		m.CapabilitiesData = m.Capabilities.Data
	}
}

// MaxAlgoRequestBytes returns max file request bytes
func (s *AlgorithmPackageService) MaxAlgoRequestBytes() int64 {
	// Add 10MB buffer for multipart metadata
	return s.opts.MaxAlgoFileSizeBytes + 10<<20
}

// UploadAndProcess streams zip file, validates, repacks to tar, registers in DB, and triggers self-check.
func (s *AlgorithmPackageService) UploadAndProcess(ctx context.Context, mr *multipart.Reader) (*model.AlgorithmPackage, error) {
	var zipPath string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			zap.L().Warn("[AlgorithmPackage] NextPart failed", zap.Error(err))
			return nil, apperrors.New(apperrors.ErrBadRequest, "")
		}

		if part.FormName() == "file" {
			// Stream part content to temporary file
			tempFile, err := os.CreateTemp("", "algo-upload-*.zip")
			if err != nil {
				part.Close()
				return nil, apperrors.New(apperrors.ErrInternal, "")
			}
			zipPath = tempFile.Name()

			_, err = io.Copy(tempFile, part)
			tempFile.Close()
			part.Close()
			if err != nil {
				os.Remove(zipPath)
				// io.Copy 失败通常因为请求体超出 MaxBytesReader 限制
				return nil, apperrors.New(apperrors.ErrFileTooLarge, "")
			}
			break
		}
		part.Close()
	}

	if zipPath == "" {
		return nil, apperrors.New(apperrors.ErrBadRequest, "")
	}

	// Defer cleanup of temporary zip file
	defer os.Remove(zipPath)

	// 确保目标存储目录存在
	tarDir := filepath.Join(s.opts.LocalUploadDir, "algorithms")
	if err := os.MkdirAll(tarDir, 0755); err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	// Define temporary name for the .tar archive
	tarFilename := fmt.Sprintf("%s.tar", uuid.New().String())
	tarPath := filepath.Join(tarDir, tarFilename)

	// Repack Zip to Tar
	meta, size, md5sum, err := RepackZipToTar(zipPath, tarPath)
	if err != nil {
		zap.L().Warn("[AlgorithmPackage] RepackZipToTar failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrBadRequest, "")
	}

	// Generate and save model
	item := &model.AlgorithmPackage{
		AlgorithmName:     meta.AlgorithmName,
		AlgorithmAlias:    meta.AlgorithmAlias,
		Version:           meta.Version,
		Domain:            meta.Domain,
		ResultSchema:      meta.ResultSchema,
		CapabilitiesImage: pq.StringArray(meta.CapabilitiesImage),
		CapabilitiesData:  pq.StringArray(meta.CapabilitiesData),
		Hardware:          pq.StringArray(meta.Hardware),
		Description:       meta.Description,
		PackagePath:       tarPath,
		ExtractPath:       tarPath,
		PackageSize:       size,
		PackageMD5:        md5sum,
		SoPath:            fmt.Sprintf("%s.so", meta.AlgorithmName),
		SelfCheckStatus:   model.SelfCheckStatusPending,
		Status:            model.AlgoPackageStatusDraft,
	}

	if meta.AIParamsSchema != nil {
		jsonBytes, err := json.Marshal(meta.AIParamsSchema)
		if err == nil {
			item.AIParamsSchema = datatypes.JSON(jsonBytes)
		}
	}

	if err := s.algorithmpackageRepo.Create(ctx, item); err != nil {
		os.Remove(tarPath)
		return nil, err
	}

	// Task 3.3: Trigger self-check
	token := uuid.New().String()
	redisKey := fmt.Sprintf("algo_token:%s", token)
	if s.rdb != nil {
		s.rdb.Set(ctx, redisKey, tarPath, 15*time.Minute)
	}

	// Format download URL — 自检接口属于 API 路由，不属于 /uploads 静态资源路径。
	downloadURL := buildAlgoDownloadURL(s.opts.PublicURL, token)

	// Trigger self check command to C++ engine
	if s.engine != nil {
		algoID := item.ID
		algoName := item.AlgorithmName
		algoVersion := item.Version

		go func() {
			defer func() {
				if r := recover(); r != nil {
					zap.L().Error("[AlgorithmPackage] self-check goroutine panicked",
						zap.Any("panic", r),
						zap.String("algo", algoName),
					)
				}
			}()

			// 通过 ID 重新查询，避免与调用者共享指针
			fresh, err := s.algorithmpackageRepo.FindByID(context.Background(), algoID)
			if err != nil {
				zap.L().Error("[AlgorithmPackage] failed to find algo for self-check",
					zap.Error(err), zap.String("algo", algoName))
				return
			}

			fresh.SelfCheckStatus = model.SelfCheckStatusRunning
			if err := s.algorithmpackageRepo.Update(context.Background(), fresh); err != nil {
				zap.L().Error("[AlgorithmPackage] failed to update self-check status to running",
					zap.Error(err), zap.String("algo", algoName))
			}

			err = s.engine.StartSelfCheck(context.Background(), downloadURL, token, algoName, algoVersion)
			now := time.Now()
			fresh.SelfCheckAt = &now
			if err != nil {
				fresh.SelfCheckStatus = model.SelfCheckStatusFailed
				fresh.SelfCheckResult = datatypes.JSON(fmt.Sprintf(`{"error": %q}`, err.Error()))
				zap.L().Error("[AlgorithmPackage] self-check failed",
					zap.Error(err), zap.String("algo", algoName))
			} else {
				fresh.SelfCheckStatus = model.SelfCheckStatusPassed
				fresh.SelfCheckResult = datatypes.JSON(`{"status": "success"}`)
				zap.L().Info("[AlgorithmPackage] self-check passed",
					zap.String("algo", algoName))
				s.enqueueFaceEmbeddingRebuild(context.Background(), algoName, algoVersion)
			}
			if err := s.algorithmpackageRepo.Update(context.Background(), fresh); err != nil {
				zap.L().Error("[AlgorithmPackage] failed to persist self-check result",
					zap.Error(err), zap.String("algo", algoName))
			}
		}()
	}

	return item, nil
}

func (s *AlgorithmPackageService) enqueueFaceEmbeddingRebuild(ctx context.Context, algoName, algoVersion string) {
	if s.taskClient == nil || algoName != defaultFaceRecognitionAlgorithm {
		return
	}
	if err := s.taskClient.Enqueue(ctx, personEmbeddingRebuildTaskType, map[string]string{
		"algo_name":    algoName,
		"algo_version": algoVersion,
	}); err != nil {
		zap.L().Warn("[AlgorithmPackage] enqueue face embedding rebuild failed",
			zap.String("algo", algoName),
			zap.String("version", algoVersion),
			zap.Error(err),
		)
		return
	}
	zap.L().Info("[AlgorithmPackage] face embedding rebuild enqueued",
		zap.String("algo", algoName),
		zap.String("version", algoVersion),
	)
}

// RepackZipToTar checks zip entries for Zip Slip, extracts meta and .so, and writes a flat .tar file.
func RepackZipToTar(zipPath, tarPath string) (*AlgoMeta, int64, string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("invalid zip file: %w", err)
	}
	defer zr.Close()

	var meta *AlgoMeta
	var foundSo bool

	// 1. First pass: Validate Zip Slip and extract metadata
	for _, f := range zr.File {
		cleanedPath := filepath.Clean(f.Name)
		if filepath.IsAbs(cleanedPath) || strings.HasPrefix(cleanedPath, "..") || strings.Contains(cleanedPath, "/../") {
			return nil, 0, "", fmt.Errorf("zip slip security exception: %s", f.Name)
		}

		if filepath.Base(cleanedPath) == "algo_meta.yaml" {
			rc, err := f.Open()
			if err != nil {
				return nil, 0, "", fmt.Errorf("open algo_meta.yaml failed: %w", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, 0, "", fmt.Errorf("read algo_meta.yaml failed: %w", err)
			}
			var m AlgoMeta
			if err := yaml.Unmarshal(data, &m); err != nil {
				return nil, 0, "", fmt.Errorf("parse algo_meta.yaml failed: %w", err)
			}
			m.Normalize()
			meta = &m
		}

		if strings.HasSuffix(filepath.Base(cleanedPath), ".so") {
			foundSo = true
		}
	}

	if meta == nil {
		return nil, 0, "", fmt.Errorf("algo_meta.yaml not found in package")
	}
	if !foundSo {
		return nil, 0, "", fmt.Errorf(".so file not found in package")
	}
	if meta.AlgorithmName == "" || meta.Version == "" {
		return nil, 0, "", fmt.Errorf("invalid algo_meta.yaml: name and version are required")
	}

	// 2. Second pass: Create tar and write entries
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("create tar file failed: %w", err)
	}
	defer func() {
		tarFile.Close()
		if err != nil {
			os.Remove(tarPath)
		}
	}()

	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	for _, f := range zr.File {
		cleanedPath := filepath.Clean(f.Name)
		rc, err := f.Open()
		if err != nil {
			return nil, 0, "", fmt.Errorf("open zip entry %s failed: %w", f.Name, err)
		}

		info := f.FileInfo()
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			rc.Close()
			return nil, 0, "", fmt.Errorf("create tar header for %s failed: %w", f.Name, err)
		}
		hdr.Name = cleanedPath

		if err := tw.WriteHeader(hdr); err != nil {
			rc.Close()
			return nil, 0, "", fmt.Errorf("write tar header for %s failed: %w", f.Name, err)
		}

		if !info.IsDir() {
			_, err = io.Copy(tw, rc)
		}
		rc.Close()
		if err != nil {
			return nil, 0, "", fmt.Errorf("copy zip entry %s to tar failed: %w", f.Name, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, 0, "", fmt.Errorf("close tar writer failed: %w", err)
	}

	fi, err := tarFile.Stat()
	if err != nil {
		return nil, 0, "", fmt.Errorf("stat tar file failed: %w", err)
	}

	// Calculate MD5
	_, _ = tarFile.Seek(0, 0)
	h := md5.New()
	if _, err := io.Copy(h, tarFile); err != nil {
		return nil, 0, "", fmt.Errorf("calculate md5 failed: %w", err)
	}
	md5sum := hex.EncodeToString(h.Sum(nil))

	return meta, fi.Size(), md5sum, nil
}

// GetPathByToken retrieves the tar path associated with the short-lived token from Redis.
// 使用后立即删除 Token，实现一次性消费，防止重复下载。
func (s *AlgorithmPackageService) GetPathByToken(ctx context.Context, token string) (string, error) {
	if s.rdb == nil {
		return "", fmt.Errorf("redis client is not initialized")
	}
	redisKey := fmt.Sprintf("algo_token:%s", token)
	val, err := s.rdb.GetDel(ctx, redisKey).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func buildAlgoDownloadURL(publicURL, token string) string {
	base := strings.TrimRight(publicURL, "/")
	if parsed, err := url.Parse(base); err == nil {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/uploads")
		parsed.RawQuery = ""
		parsed.Fragment = ""
		base = strings.TrimRight(parsed.String(), "/")
	}
	return fmt.Sprintf("%s/api/v1/internal/algo/download?token=%s", base, url.QueryEscape(token))
}
