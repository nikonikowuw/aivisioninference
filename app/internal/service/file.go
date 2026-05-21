package service

import (
	"context"
	"crypto/md5"
	"encoding/json"
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

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
	"github.com/niko-admin/niko-admin/internal/repository"
)

const chunkDir = "tmp/uploads"

// FileService 处理文件分片上传、合并、下载及删除相关的业务逻辑
type FileService struct {
	fileRepo *repository.FileRepository // 文件数据持久化接口
}

// NewFileService 创建并返回一个新的 FileService 实例
func NewFileService(fileRepo *repository.FileRepository) *FileService {
	return &FileService{fileRepo: fileRepo}
}

// InitUpload 初始化一个分片上传会话，创建临时目录并保存分片元数据
func (s *FileService) InitUpload(ctx context.Context, req dto.InitUploadRequest) (*model.FileChunk, error) {
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

	if _, err := io.Copy(dst, chunkData); err != nil {
		zap.L().Error("write chunk file failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	// 更新已上传的分片索引列表，支持幂等上传：网络抖动导致客户端重试同一分片时，
	// 已存在的 index 不会重复记录，避免 CompleteUpload 误判分片完整性。
	var uploaded []int
	if err := json.Unmarshal([]byte(chunk.UploadedChunks), &uploaded); err != nil {
		uploaded = []int{}
	}

	found := false
	for _, u := range uploaded {
		if u == index {
			found = true
			break
		}
	}
	if !found {
		uploaded = append(uploaded, index)
		sort.Ints(uploaded)
	}

	data, _ := json.Marshal(uploaded)
	if err := s.fileRepo.UpdateChunkUploadedChunks(ctx, uploadID, string(data)); err != nil {
		zap.L().Error("update uploaded chunks failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}

// CompleteUpload 合并所有已上传分片，验证 MD5 校验和，移动到正式上传目录并记录文件记录
func (s *FileService) CompleteUpload(ctx context.Context, uploadID string) (*model.File, error) {
	chunk, err := s.fileRepo.FindByUploadIDWithStatus(ctx, uploadID, "uploading")
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "上传会话不存在或已过期")
	}

	// 使用 canceled 标记做两阶段清理：仅在上下文取消时删除临时分片数据，
	// 正常流程不应清理（文件已经合并完毕移动到正式目录）。
	// cleanup 使用 context.WithoutCancel 剥离原始 ctx 的取消信号，
	// 确保 deferred 清理操作不受父上下文取消影响（父 ctx 取消时清理也必须执行）。
	canceled := false
	defer func() {
		if canceled {
			if err := s.fileRepo.DeleteChunkByUploadID(context.WithoutCancel(ctx), uploadID); err != nil {
				zap.L().Warn("cleanup canceled upload chunk record failed", zap.String("upload_id", uploadID), zap.Error(err))
			}
			if err := os.RemoveAll(filepath.Join(chunkDir, uploadID)); err != nil {
				zap.L().Warn("cleanup canceled upload chunk dir failed", zap.String("upload_id", uploadID), zap.Error(err))
			}
		}
	}()

	// 验证所有分片是否已上传，JSON 格式的已上传索引列表应与总分片数一致。
	var uploaded []int
	if err := json.Unmarshal([]byte(chunk.UploadedChunks), &uploaded); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "分片状态异常")
	}
	if len(uploaded) != chunk.TotalChunks {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("分片不完整，已上传 %d/%d", len(uploaded), chunk.TotalChunks))
	}

	chunkDirPath := filepath.Join(chunkDir, uploadID)
	mergedPath := filepath.Join(chunkDirPath, "merged")

	// 在各关键步骤间检查上下文是否已取消，实现优雅中断。
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

	// 对整个合并后的文件做 MD5 校验，确保所有分片还原正确。
	mergedData, err := os.ReadFile(mergedPath)
	if err != nil {
		zap.L().Error("read merged file for md5 failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	actualMD5 := fmt.Sprintf("%x", md5.Sum(mergedData))
	if actualMD5 != chunk.MD5 {
		os.RemoveAll(chunkDirPath)
		if err := s.fileRepo.UpdateChunkStatus(ctx, uploadID, "failed"); err != nil {
			zap.L().Warn("mark chunk upload failed", zap.String("upload_id", uploadID), zap.Error(err))
		}
		return nil, apperrors.New(apperrors.ErrBadRequest, "文件校验失败（MD5 不匹配）")
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
		MimeType:     detectMimeType(chunk.FileName),
		Size:         chunk.FileSize,
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

// GetUploadProgress 获取并返回上传会话的已上传分片索引列表及总分片数
func (s *FileService) GetUploadProgress(ctx context.Context, uploadID string) (*dto.UploadProgressResponse, error) {
	chunk, err := s.fileRepo.FindByUploadID(ctx, uploadID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "上传会话不存在")
	}

	var uploaded []int
	if err := json.Unmarshal([]byte(chunk.UploadedChunks), &uploaded); err != nil {
		uploaded = []int{}
	}

	return &dto.UploadProgressResponse{
		UploadID:       chunk.UploadID,
		UploadedChunks: uploaded,
		TotalChunks:    chunk.TotalChunks,
	}, nil
}

// CheckFile 检查文件是否已存在（用于秒传，当前作为占位功能，固定返回不存在）
func (s *FileService) CheckFile(ctx context.Context, md5Hash string) (*dto.CheckFileResponse, error) {
	return &dto.CheckFileResponse{Exists: false}, nil
}

// List returns a paginated list of files with optional filters.
//
// 【核心功能】查询文件列表，支持关键字、存储类型、时间范围等筛选条件。
// 时间范围参数通过公共 ParseTimeRange 方法解析，确保格式统一。
func (s *FileService) List(ctx context.Context, req dto.FileListRequest) ([]model.File, int64, error) {
	// 解析时间范围参数
	if req.StartTime != "" || req.EndTime != "" {
		from, to, err := scopes.ParseTimeRange(req.StartTime, req.EndTime)
		if err != nil {
			return nil, 0, mapTimeRangeError(err)
		}
		req.FromTime = from
		req.ToTime = to
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

	var start, end int64
	var err error

	if splits[0] == "" {
		// 后缀范围: bytes=-500 表示文件最后 500 个字节
		end = fileSize - 1
		suffixLen, err := strconv.ParseInt(splits[1], 10, 64)
		if err != nil || suffixLen <= 0 {
			return 0, 0, false
		}
		start = fileSize - suffixLen
		if start < 0 {
			start = 0
		}
	} else if splits[1] == "" {
		// 前缀范围: bytes=0- 表示从字节 0 到文件末尾
		start, err = strconv.ParseInt(splits[0], 10, 64)
		if err != nil || start < 0 || start >= fileSize {
			return 0, 0, false
		}
		end = fileSize - 1
	} else {
		// 完整范围: bytes=0-499 表示第 0 到 499 字节
		start, err = strconv.ParseInt(splits[0], 10, 64)
		if err != nil || start < 0 {
			return 0, 0, false
		}
		end, err = strconv.ParseInt(splits[1], 10, 64)
		if err != nil || end < start || end >= fileSize {
			return 0, 0, false
		}
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

// detectMimeType 根据文件名后缀名映射其相应的 MIME 类型
func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeMap := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
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
	}
	if mime, ok := mimeMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
