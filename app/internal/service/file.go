// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/csvx"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/timex"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// chunkDir 分片上传的临时存储目录。
const chunkDir = "tmp/uploads"

// FileOptions 定义文件上传安全限制。
type FileOptions struct {
	MaxFileSizeBytes  int64
	MaxChunkSizeBytes int64
}

// FileService 处理文件分片上传、合并、下载及删除相关的业务逻辑
type FileService struct {
	fileRepo *repository.FileRepository // 文件数据持久化接口
	opts     FileOptions                // 上传大小限制配置
}

// NewFileService 创建并返回一个新的 FileService 实例，可选的 opts 参数用于覆盖默认上传限制。
func NewFileService(fileRepo *repository.FileRepository, opts ...FileOptions) *FileService {
	option := FileOptions{
		MaxFileSizeBytes:  100 << 20,
		MaxChunkSizeBytes: 5 << 20,
	}
	if len(opts) > 0 {
		option = opts[0]
	}
	return &FileService{fileRepo: fileRepo, opts: option}
}

// MaxChunkSizeBytes 返回单分片最大字节数，供 Handler 限制 multipart 内存阈值。
func (s *FileService) MaxChunkSizeBytes() int64 {
	return s.opts.MaxChunkSizeBytes
}

// MaxChunkRequestBytes 返回单次分片上传请求体最大字节数，包含 multipart 边界和普通字段开销。
func (s *FileService) MaxChunkRequestBytes() int64 {
	const multipartOverheadBytes int64 = 1 << 20
	if s.opts.MaxChunkSizeBytes <= 0 {
		return 0
	}
	return s.opts.MaxChunkSizeBytes + multipartOverheadBytes
}

// InitUpload 初始化一个分片上传会话，创建临时目录并保存分片元数据
func (s *FileService) InitUpload(ctx context.Context, req dto.InitUploadRequest) (*model.FileChunk, error) {
	if s.opts.MaxFileSizeBytes > 0 && req.FileSize > s.opts.MaxFileSizeBytes {
		return nil, apperrors.New(apperrors.ErrBadRequest, "文件大小超过限制")
	}
	if isSVGFile(req.FileName) {
		return nil, apperrors.New(apperrors.ErrBadRequest, "不支持上传 SVG 文件")
	}
	if s.opts.MaxChunkSizeBytes > 0 {
		expectedChunks := int((req.FileSize + s.opts.MaxChunkSizeBytes - 1) / s.opts.MaxChunkSizeBytes)
		if req.TotalChunks != expectedChunks {
			return nil, apperrors.New(apperrors.ErrBadRequest, "分片数量与文件大小不匹配")
		}
	}

	storageType := req.StorageType
	if storageType == "" {
		storageType = "local"
	}

	chunk := model.FileChunk{
		UploadID:       uuid.New().String(),
		FileName:       req.FileName,
		FileSize:       req.FileSize,
		MD5:            req.MD5,
		TotalChunks:    req.TotalChunks,
		Status:         "uploading",
		StorageType:    storageType,
		UploadedChunks: "[]",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	if err := s.fileRepo.CreateChunk(ctx, &chunk); err != nil {
		zap.L().Error("init upload failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	dir := filepath.Join(chunkDir, chunk.UploadID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		zap.L().Error("create chunk dir failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	return &chunk, nil
}

// SaveChunk 将单个上传的分片保存到临时目录中，并更新已上传的分片索引列表
func (s *FileService) SaveChunk(ctx context.Context, uploadID string, index int, chunkData io.Reader) error {
	chunk, err := s.fileRepo.FindByUploadIDWithStatus(ctx, uploadID, "uploading")
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "上传会话不存在或已过期")
	}

	if index < 0 || index >= chunk.TotalChunks {
		return apperrors.New(apperrors.ErrBadRequest, "无效的分片索引")
	}

	// 将分片写入临时目录，文件名为 chunk_{index}，与后续 merge 逻辑匹配。
	chunkPath := filepath.Join(chunkDir, uploadID, fmt.Sprintf("chunk_%d", index))
	dst, err := os.Create(chunkPath)
	if err != nil {
		zap.L().Error("create chunk file failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}
	defer dst.Close()

	written, err := copyWithLimit(dst, chunkData, s.opts.MaxChunkSizeBytes)
	if err != nil {
		zap.L().Error("write chunk file failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}
	if s.opts.MaxChunkSizeBytes > 0 && written > s.opts.MaxChunkSizeBytes {
		if removeErr := os.Remove(chunkPath); removeErr != nil {
			zap.L().Warn("remove oversized chunk failed", zap.String("path", chunkPath), zap.Error(removeErr))
		}
		return apperrors.New(apperrors.ErrBadRequest, "分片大小超过限制")
	}

	uploaded := s.unmarshalUploadedChunks(chunk.UploadedChunks)
	uploaded = appendUniqueChunkIndex(uploaded, index)

	data, _ := json.Marshal(uploaded)
	if err := s.fileRepo.UpdateChunkUploadedChunks(ctx, uploadID, string(data)); err != nil {
		zap.L().Error("update uploaded chunks failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}

// unmarshalUploadedChunks 将已上传分片索引的 JSON 字符串解析为整型切片。
func (s *FileService) unmarshalUploadedChunks(data string) []int {
	var uploaded []int
	_ = json.Unmarshal([]byte(data), &uploaded)
	return uploaded
}

// CompleteUpload 合并所有已上传分片，验证 MD5 校验和，移动到正式上传目录并记录文件记录
func (s *FileService) CompleteUpload(ctx context.Context, uploadID string) (*model.File, error) {
	chunk, err := s.fileRepo.FindByUploadIDWithStatus(ctx, uploadID, "uploading")
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "上传会话不存在或已过期")
	}

	canceled := false
	defer func() {
		if canceled {
			s.cleanupCanceledUpload(ctx, uploadID)
		}
	}()

	uploaded := s.unmarshalUploadedChunks(chunk.UploadedChunks)
	if len(uploaded) != chunk.TotalChunks {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("分片不完整，已上传 %d/%d", len(uploaded), chunk.TotalChunks))
	}

	chunkDirPath := filepath.Join(chunkDir, uploadID)
	mergedPath := filepath.Join(chunkDirPath, "merged")

	if err := ctx.Err(); err != nil {
		canceled = true
		return nil, apperrors.New(apperrors.ErrBadRequest, "上传已取消")
	}

	mergedFile, err := os.Create(mergedPath)
	if err != nil {
		zap.L().Error("create merged file failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	defer mergedFile.Close()

	// 按索引顺序拼接所有分片，保证合并后文件内容与原始文件一致。
	for i := 0; i < chunk.TotalChunks; i++ {
		if err := ctx.Err(); err != nil {
			canceled = true
			return nil, apperrors.New(apperrors.ErrBadRequest, "上传已取消")
		}
		chunkPath := filepath.Join(chunkDirPath, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			zap.L().Error("open chunk file failed", zap.Int("index", i), zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		if _, err := io.Copy(mergedFile, chunkFile); err != nil {
			chunkFile.Close()
			if ctx.Err() != nil {
				canceled = true
			}
			zap.L().Error("merge chunk failed", zap.Int("index", i), zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		chunkFile.Close()
	}
	mergedFile.Close()

	if err := ctx.Err(); err != nil {
		canceled = true
		return nil, apperrors.New(apperrors.ErrBadRequest, "上传已取消")
	}

	mergedInfo, err := os.Stat(mergedPath)
	if err != nil {
		zap.L().Error("stat merged file failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if mergedInfo.Size() != chunk.FileSize {
		s.markUploadFailed(ctx, uploadID, chunkDirPath)
		return nil, apperrors.New(apperrors.ErrBadRequest, "文件大小与声明不一致")
	}

	// 对整个合并后的文件做流式 MD5 校验，避免大文件一次性读入内存。
	actualMD5, err := fileMD5(mergedPath)
	if err != nil {
		zap.L().Error("calculate merged file md5 failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if actualMD5 != chunk.MD5 {
		s.markUploadFailed(ctx, uploadID, chunkDirPath)
		return nil, apperrors.New(apperrors.ErrBadRequest, "文件校验失败（MD5 不匹配）")
	}

	// 增强安全性：基于合并后的文件内容探测 MIME 类型，防止后缀名欺骗。
	mimeType := detectMimeType(chunk.FileName, mergedPath)
	if mimeType == "image/svg+xml" {
		s.markUploadFailed(ctx, uploadID, chunkDirPath)
		return nil, apperrors.New(apperrors.ErrBadRequest, "不支持上传 SVG 文件")
	}

	// 按日期分目录存储，避免单个目录中文件过多影响性能。
	dateDir := time.Now().Format("2006/01/02")
	ext := filepath.Ext(chunk.FileName)
	storageName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	storagePath := filepath.Join(dateDir, storageName)

	finalDir := filepath.Join("uploads", dateDir)
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		zap.L().Error("create storage dir failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	finalPath := filepath.Join("uploads", storagePath)
	// os.Rename 要求源和目标在同一文件系统分区，临时目录和 uploads 目录在设计上位于同一分区。
	if err := os.Rename(mergedPath, finalPath); err != nil {
		zap.L().Error("move merged file failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	// 合并成功后删除临时分片目录，释放磁盘空间。
	os.RemoveAll(chunkDirPath)

	fileRecord := model.File{
		Name:         storageName,
		OriginalName: chunk.FileName,
		Path:         finalPath,
		MimeType:     mimeType,
		Size:         chunk.FileSize,
		MD5:          chunk.MD5,
		StorageType:  chunk.StorageType,
	}

	if err := s.fileRepo.Create(ctx, &fileRecord); err != nil {
		zap.L().Error("create file record failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	// 标记分片上传会话为已完成。
	now := time.Now()
	if err := s.fileRepo.UpdateChunkCompleted(ctx, uploadID, &now); err != nil {
		zap.L().Warn("mark chunk upload completed", zap.String("upload_id", uploadID), zap.Error(err))
	}

	return &fileRecord, nil
}

// markUploadFailed 标记上传失败并清理临时目录。
func (s *FileService) markUploadFailed(ctx context.Context, uploadID, chunkDirPath string) {
	_ = os.RemoveAll(chunkDirPath)
	if err := s.fileRepo.UpdateChunkStatus(ctx, uploadID, "failed"); err != nil {
		zap.L().Warn("mark chunk upload failed", zap.String("upload_id", uploadID), zap.Error(err))
	}
}

// cleanupCanceledUpload 清理已取消上传的数据库记录和临时目录。
func (s *FileService) cleanupCanceledUpload(ctx context.Context, uploadID string) {
	cleanupCtx := context.WithoutCancel(ctx)
	if err := s.fileRepo.DeleteChunkByUploadID(cleanupCtx, uploadID); err != nil {
		zap.L().Warn("cleanup canceled upload chunk record failed", zap.String("upload_id", uploadID), zap.Error(err))
	}
	if err := os.RemoveAll(filepath.Join(chunkDir, uploadID)); err != nil {
		zap.L().Warn("cleanup canceled upload chunk dir failed", zap.String("upload_id", uploadID), zap.Error(err))
	}
}

// GetUploadProgress 获取并返回上传会话的已上传分片索引列表及总分片数
func (s *FileService) GetUploadProgress(ctx context.Context, uploadID string) (*dto.UploadProgressResponse, error) {
	chunk, err := s.fileRepo.FindByUploadID(ctx, uploadID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "上传会话不存在")
	}

	return &dto.UploadProgressResponse{
		UploadID:       chunk.UploadID,
		UploadedChunks: s.unmarshalUploadedChunks(chunk.UploadedChunks),
		TotalChunks:    chunk.TotalChunks,
	}, nil
}

// CheckFile 检查文件是否已存在（用于秒传）。
func (s *FileService) CheckFile(ctx context.Context, md5Hash string) (*dto.CheckFileResponse, error) {
	file, err := s.fileRepo.FindByMD5(ctx, md5Hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &dto.CheckFileResponse{Exists: false}, nil
		}
		zap.L().Error("check file by md5 failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return &dto.CheckFileResponse{Exists: true, FileID: file.ID}, nil
}

// List 查询文件列表，支持关键字、存储类型、时间范围等筛选条件。
func (s *FileService) List(ctx context.Context, req dto.FileListRequest) ([]model.File, int64, error) {
	if err := normalizeFileTimeRange(&req); err != nil {
		return nil, 0, err
	}
	return s.fileRepo.List(ctx, req)
}

// GetByID 根据文件 ID 查询文件信息
func (s *FileService) GetByID(ctx context.Context, id string) (*model.File, error) {
	file, err := s.fileRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "文件不存在")
	}
	return file, nil
}

// DownloadInfo 包含下载文件所需的所有元数据与文件系统信息
type DownloadInfo struct {
	File        *model.File // 文件模型实例
	FilePath    string      // 磁盘物理文件路径
	FileSize    int64       // 物理文件实际大小
	ContentType string      // 文件 MIME 类型
	RangeHeader string      // HTTP Range 请求头内容
}

// GetDownloadInfo 准备下载文件所需的信息并验证物理文件在磁盘上的存在性
func (s *FileService) GetDownloadInfo(ctx context.Context, id string, rangeHeader string) (*DownloadInfo, error) {
	file, err := s.fileRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "文件不存在")
	}

	fileInfo, err := os.Stat(file.Path)
	if err != nil {
		zap.L().Error("file not found on disk", zap.String("id", id), zap.String("path", file.Path), zap.Error(err))
		return nil, apperrors.New(apperrors.ErrNotFound, "文件已丢失")
	}

	contentType := file.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return &DownloadInfo{
		File:        file,
		FilePath:    file.Path,
		FileSize:    fileInfo.Size(),
		ContentType: contentType,
		RangeHeader: rangeHeader,
	}, nil
}

// Delete 软删除文件数据库记录，并尽力而为地删除磁盘上的物理文件
func (s *FileService) Delete(ctx context.Context, id string) error {
	file, err := s.fileRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "文件不存在")
	}

	if err := s.fileRepo.Delete(ctx, id); err != nil {
		zap.L().Error("delete file record failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	// Best-effort 清理物理文件：数据库记录已删除，物理文件清理失败不应阻塞业务响应。
	if file.Path != "" {
		if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
			zap.L().Warn("remove physical file failed", zap.String("id", id), zap.String("path", file.Path), zap.Error(err))
		}
	}

	return nil
}

// BatchDelete 批量删除文件，逐条复用单条删除逻辑。
func (s *FileService) BatchDelete(ctx context.Context, ids []string, lang string) dto.BatchResult {
	return runBatch(ids, lang, func(id string) error {
		return s.Delete(ctx, id)
	})
}

// ExportCSV 导出当前筛选条件下的文件列表 CSV。
func (s *FileService) ExportCSV(ctx context.Context, req dto.FileListRequest) ([]byte, error) {
	if err := normalizeFileTimeRange(&req); err != nil {
		return nil, err
	}
	items, err := s.fileRepo.ListForExport(ctx, req, maxCSVExportRows)
	if err != nil {
		zap.L().Error("export files failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID,
			item.OriginalName,
			item.MimeType,
			strconv.FormatInt(item.Size, 10),
			item.StorageType,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	data, err := csvx.Build([]string{"ID", "OriginalName", "MimeType", "Size", "StorageType", "CreatedAt"}, rows)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return data, nil
}

func normalizeFileTimeRange(req *dto.FileListRequest) error {
	return timex.NormalizeRange(req.StartTime, req.EndTime, &req.FromTime, &req.ToTime)
}

// ParseRange parses the Range header value and returns (start, end, ok).
// 实现 RFC 7233 §2.1 定义的三种 Range 格式，用于断点续传支持。
func ParseRange(rangeHeader string, fileSize int64) (int64, int64, bool) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, false
	}
	parts := strings.TrimPrefix(rangeHeader, "bytes=")
	splits := strings.SplitN(parts, "-", 2)
	if len(splits) != 2 {
		return 0, 0, false
	}

	switch {
	case splits[0] == "":
		suffixLen, err := strconv.ParseInt(splits[1], 10, 64)
		if err != nil || suffixLen <= 0 {
			return 0, 0, false
		}
		return max(fileSize-suffixLen, 0), fileSize - 1, true
	case splits[1] == "":
		start, err := strconv.ParseInt(splits[0], 10, 64)
		if err != nil || start < 0 || start >= fileSize {
			return 0, 0, false
		}
		return start, fileSize - 1, true
	}

	start, err := strconv.ParseInt(splits[0], 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	end, err := strconv.ParseInt(splits[1], 10, 64)
	if err != nil || end < start || end >= fileSize {
		return 0, 0, false
	}

	return start, end, true
}

// WriteRange 将物理文件指定字节区间的内容写入 http.ResponseWriter 并设置正确的 Range 响应头
func WriteRange(w http.ResponseWriter, filePath string, start, end, totalSize int64, contentType string) error {
	length := end - start + 1
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusPartialContent)

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return err
	}
	_, err = io.CopyN(w, file, length)
	return err
}

// detectMimeType 根据文件名和文件内容探测 MIME 类型。
func detectMimeType(filename, path string) string {
	if file, err := os.Open(path); err == nil {
		defer file.Close()
		buffer := make([]byte, 512)
		if n, _ := file.Read(buffer); n > 0 {
			contentType := http.DetectContentType(buffer[:n])
			if contentType != "application/octet-stream" && contentType != "text/plain" {
				return contentType
			}
		}
	}

	ext := strings.ToLower(filepath.Ext(filename))
	mimeMap := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".mp4":  "video/mp4",
		".mp3":  "audio/mpeg",
		".zip":  "application/zip",
		".txt":  "text/plain",
		".json": "application/json",
		".csv":  "text/csv",
		".svg":  "image/svg+xml",
	}
	if mime, ok := mimeMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// appendUniqueChunkIndex 追加未记录的分片索引，并保持索引列表有序。
func appendUniqueChunkIndex(uploaded []int, index int) []int {
	for _, uploadedIndex := range uploaded {
		if uploadedIndex == index {
			return uploaded
		}
	}
	uploaded = append(uploaded, index)
	sort.Ints(uploaded)
	return uploaded
}

// copyWithLimit 写入最多 limit+1 字节，用于识别超过限制的分片。
func copyWithLimit(dst io.Writer, src io.Reader, limit int64) (int64, error) {
	if limit <= 0 {
		return io.Copy(dst, src)
	}
	return io.Copy(dst, io.LimitReader(src, limit+1))
}

// fileMD5 使用流式读取计算文件 MD5，避免大文件占用大量内存。
func fileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// isSVGFile 根据扩展名拒绝 SVG，避免静态目录下产生存储型 XSS 风险。
func isSVGFile(filename string) bool {
	return strings.EqualFold(filepath.Ext(filename), ".svg")
}
