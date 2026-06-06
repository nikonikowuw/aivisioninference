// Package task 提供人员批量导入的异步任务处理器。
package task

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

// PersonImportPayload 导入任务载荷。
type PersonImportPayload struct {
	TaskID    string `json:"task_id"`
	FileURL   string `json:"file_url"`
	Overwrite bool   `json:"overwrite"`
}

// PersonImportHandler 处理人员批量导入任务。
type PersonImportHandler struct {
	personRepo     *repository.PersonRepository
	embeddingRepo  *repository.PersonEmbeddingRepository
	importTaskRepo *repository.ImportTaskRepository
	storage        storage.Storage
	taskClient     *Client
}

// NewPersonImportHandler 创建导入任务处理器。
func NewPersonImportHandler(
	personRepo *repository.PersonRepository,
	embeddingRepo *repository.PersonEmbeddingRepository,
	importTaskRepo *repository.ImportTaskRepository,
	storage storage.Storage,
	taskClient *Client,
) *PersonImportHandler {
	return &PersonImportHandler{
		personRepo:     personRepo,
		embeddingRepo:  embeddingRepo,
		importTaskRepo: importTaskRepo,
		storage:        storage,
		taskClient:     taskClient,
	}
}

// RegisterHandlers 注册处理器。
func (h *PersonImportHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypePersonImport, h.handleImport)
}

// supportedImageExts 支持的图片扩展名集合。
var supportedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

// archiveEntry 表示压缩包内的一个文件条目。
type archiveEntry struct {
	Name string // 文件完整路径（在压缩包内）
	Open func() (io.ReadCloser, error)
}

// handleImport 处理批量导入：从压缩包中读取图片，以文件名作为人员姓名（或编号_姓名格式）。
func (h *PersonImportHandler) handleImport(ctx context.Context, t *asynq.Task) error {
	var payload PersonImportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	task, err := h.importTaskRepo.FindByID(ctx, payload.TaskID)
	if err != nil {
		return fmt.Errorf("find import task: %w", err)
	}

	// 更新状态为运行中
	task.Status = model.ImportTaskStatusRunning
	_ = h.importTaskRepo.Update(ctx, task)

	// 从 Storage 读取压缩包
	reader, err := h.storage.Get(payload.FileURL)
	if err != nil {
		task.Status = model.ImportTaskStatusFailed
		_ = h.importTaskRepo.Update(ctx, task)
		return fmt.Errorf("read archive: %w", err)
	}
	defer reader.Close()

	// 完整读入内存，设置安全上限防止 OOM
	const maxArchiveSize = 100 << 20 // 100MB
	limitedReader := io.LimitReader(reader, maxArchiveSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		task.Status = model.ImportTaskStatusFailed
		_ = h.importTaskRepo.Update(ctx, task)
		return fmt.Errorf("read archive data: %w", err)
	}
	if int64(len(data)) > maxArchiveSize {
		task.Status = model.ImportTaskStatusFailed
		_ = h.importTaskRepo.Update(ctx, task)
		return fmt.Errorf("archive exceeds maximum size limit (%d bytes)", maxArchiveSize)
	}

	// 根据文件后缀选择解压方式
	entries, err := openArchive(payload.FileURL, data)
	if err != nil {
		task.Status = model.ImportTaskStatusFailed
		_ = h.importTaskRepo.Update(ctx, task)
		return fmt.Errorf("open archive: %w", err)
	}

	// 过滤出图片文件
	var images []archiveEntry
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name))
		if supportedImageExts[ext] {
			images = append(images, e)
		}
	}

	totalRows := len(images)
	successRows, failedRows := 0, 0

	for _, img := range images {
		fileName := filepath.Base(img.Name)
		baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		// 解析文件名：支持 "编号_姓名" 或纯 "姓名" 格式
		personCode, personName := parseImportFileName(baseName)
		if personName == "" {
			failedRows++
			zap.L().Warn("skip entry with empty name", zap.String("file", fileName))
			continue
		}

		// 自动生成编号（如果未从文件名解析到）
		if personCode == "" {
			personCode = "IMP" + time.Now().Format("20060102150405") + uuid.New().String()[:6]
		}

		// 检查人员编号唯一性
		if exists, _ := h.personRepo.ExistsByPersonCode(ctx, personCode, ""); exists {
			if !payload.Overwrite {
				failedRows++
				continue
			}
		}

		// 读取图片数据
		rc, err := img.Open()
		if err != nil {
			failedRows++
			zap.L().Warn("open archive entry failed", zap.String("file", fileName), zap.Error(err))
			continue
		}
		imgData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			failedRows++
			zap.L().Warn("read archive entry failed", zap.String("file", fileName), zap.Error(err))
			continue
		}

		// 保存图片到 Storage
		ext := strings.ToLower(filepath.Ext(fileName))
		storagePath := "imports/persons/" + uuid.New().String() + ext
		if _, err := h.storage.Save(bytes.NewReader(imgData), storagePath); err != nil {
			failedRows++
			zap.L().Warn("save imported image failed", zap.String("file", fileName), zap.Error(err))
			continue
		}

		imageURL := fmt.Sprintf("/api/v1/persons/image/%s", filepath.Base(storagePath))

		// 创建人员记录
		person := &model.Person{
			PersonCode:      personCode,
			PersonName:      personName,
			Gender:          model.GenderUnknown,
			ImageURL:        imageURL,
			EmbeddingStatus: model.EmbeddingStatusPending,
			Enabled:         true,
		}

		if err := h.personRepo.Create(ctx, person); err != nil {
			failedRows++
			zap.L().Warn("create person from archive import failed", zap.String("person_code", personCode), zap.Error(err))
			continue
		}

		// 投递特征提取任务
		if err := h.taskClient.Enqueue(ctx, TypePersonEmbedding, map[string]string{"person_id": person.ID}); err != nil {
			zap.L().Warn("enqueue embedding task from archive import failed", zap.String("person_id", person.ID), zap.Error(err))
		}
		successRows++
	}

	// 更新任务状态
	task.TotalRows = totalRows
	task.SuccessRows = successRows
	task.FailedRows = failedRows
	task.Status = model.ImportTaskStatusCompleted
	_ = h.importTaskRepo.Update(ctx, task)

	zap.L().Info("person import completed",
		zap.String("task_id", task.ID),
		zap.Int("total", totalRows),
		zap.Int("success", successRows),
		zap.Int("failed", failedRows),
	)
	return nil
}

// openArchive 根据文件后缀检测压缩包格式并返回文件条目列表。
// 支持格式：.zip、.tar.gz(.tgz)、.tar.bz2(.tbz2)
func openArchive(filename string, data []byte) ([]archiveEntry, error) {
	name := strings.ToLower(filename)

	// .zip
	if strings.HasSuffix(name, ".zip") {
		return openZip(data)
	}
	// .tar.gz / .tgz
	if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") {
		return openTarGzip(data)
	}
	// .tar.bz2 / .tbz2
	if strings.HasSuffix(name, ".tar.bz2") || strings.HasSuffix(name, ".tbz2") {
		return openTarBzip2(data)
	}

	return nil, fmt.Errorf("unsupported archive format: %s (supported: zip, tar.gz/tgz, tar.bz2/tbz2)", filepath.Ext(filename))
}

// openZip 解压 ZIP 格式。
func openZip(data []byte) ([]archiveEntry, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	var entries []archiveEntry
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// 使用闭包捕获当前文件引用
		file := f
		entries = append(entries, archiveEntry{
			Name: file.Name,
			Open: func() (io.ReadCloser, error) {
				return file.Open()
			},
		})
	}
	return entries, nil
}

// openTarGzip 解压 TAR.GZ 格式。
func openTarGzip(data []byte) ([]archiveEntry, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	tarData, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	return parseTar(bytes.NewReader(tarData))
}

// openTarBzip2 解压 TAR.BZ2 格式。
func openTarBzip2(data []byte) ([]archiveEntry, error) {
	br := bzip2.NewReader(bytes.NewReader(data))
	tarData, err := io.ReadAll(br)
	if err != nil {
		return nil, fmt.Errorf("read bzip2: %w", err)
	}
	return parseTar(bytes.NewReader(tarData))
}

// parseTar 从 TAR 数据中提取文件条目。
// 使用 sync.Pool 优化内存：为每个条目创建独立的 bytes.Reader。
func parseTar(r io.Reader) ([]archiveEntry, error) {
	tr := tar.NewReader(r)
	var entries []archiveEntry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// 读取条目内容到内存，以便后续多次读取
		content, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read tar entry %s: %w", hdr.Name, err)
		}
		// 拷贝内容避免引用问题
		copied := make([]byte, len(content))
		copy(copied, content)
		name := hdr.Name
		entries = append(entries, archiveEntry{
			Name: name,
			Open: func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(copied)), nil
			},
		})
	}
	return entries, nil
}

// parseImportFileName 解析导入文件名，支持以下格式：
//   - "编号_姓名" → (personCode, personName)
//   - "姓名" → ("", personName)
func parseImportFileName(baseName string) (personCode, personName string) {
	parts := strings.SplitN(baseName, "_", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", strings.TrimSpace(baseName)
}
