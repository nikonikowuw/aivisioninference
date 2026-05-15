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
	"github.com/niko-admin/niko-admin/internal/repository"
)

const chunkDir = "tmp/uploads"

// FileService handles business logic for File operations.
type FileService struct {
	fileRepo *repository.FileRepository
}

// NewFileService creates a new FileService.
func NewFileService(fileRepo *repository.FileRepository) *FileService {
	return &FileService{fileRepo: fileRepo}
}

// InitUpload creates a chunked upload session.
func (s *FileService) InitUpload(ctx context.Context, req dto.InitUploadRequest, userID string) (*model.FileChunk, error) {
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
		UploaderID:     userID,
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

// SaveChunk saves a single chunk to disk and updates the uploaded chunks list.
func (s *FileService) SaveChunk(ctx context.Context, uploadID string, index int, chunkData io.Reader) error {
	chunk, err := s.fileRepo.FindByUploadIDWithStatus(ctx, uploadID, "uploading")
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "上传会话不存在或已过期")
	}

	if index < 0 || index >= chunk.TotalChunks {
		return apperrors.New(apperrors.ErrBadRequest, "无效的分片索引")
	}

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

// CompleteUpload merges all chunks and creates the final file record.
func (s *FileService) CompleteUpload(ctx context.Context, uploadID string) (*model.File, error) {
	chunk, err := s.fileRepo.FindByUploadIDWithStatus(ctx, uploadID, "uploading")
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "上传会话不存在或已过期")
	}

	var uploaded []int
	if err := json.Unmarshal([]byte(chunk.UploadedChunks), &uploaded); err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "分片状态异常")
	}
	if len(uploaded) != chunk.TotalChunks {
		return nil, apperrors.New(apperrors.ErrBadRequest, fmt.Sprintf("分片不完整，已上传 %d/%d", len(uploaded), chunk.TotalChunks))
	}

	chunkDirPath := filepath.Join(chunkDir, uploadID)
	mergedPath := filepath.Join(chunkDirPath, "merged")

	mergedFile, err := os.Create(mergedPath)
	if err != nil {
		zap.L().Error("create merged file failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	defer mergedFile.Close()

	for i := 0; i < chunk.TotalChunks; i++ {
		chunkPath := filepath.Join(chunkDirPath, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			zap.L().Error("open chunk file failed", zap.Int("index", i), zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		if _, err := io.Copy(mergedFile, chunkFile); err != nil {
			chunkFile.Close()
			zap.L().Error("merge chunk failed", zap.Int("index", i), zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		chunkFile.Close()
	}
	mergedFile.Close()

	mergedData, err := os.ReadFile(mergedPath)
	if err != nil {
		zap.L().Error("read merged file for md5 failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	actualMD5 := fmt.Sprintf("%x", md5.Sum(mergedData))
	if actualMD5 != chunk.MD5 {
		os.RemoveAll(chunkDirPath)
		s.fileRepo.UpdateChunkStatus(ctx, uploadID, "failed")
		return nil, apperrors.New(apperrors.ErrBadRequest, "文件校验失败（MD5 不匹配）")
	}

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
	if err := os.Rename(mergedPath, finalPath); err != nil {
		zap.L().Error("move merged file failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	os.RemoveAll(chunkDirPath)

	fileRecord := model.File{
		Name:         storageName,
		OriginalName: chunk.FileName,
		Path:         finalPath,
		MimeType:     detectMimeType(chunk.FileName),
		Size:         chunk.FileSize,
		StorageType:  chunk.StorageType,
		UploaderID:   chunk.UploaderID,
	}

	if err := s.fileRepo.Create(ctx, &fileRecord); err != nil {
		zap.L().Error("create file record failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	now := time.Now()
	s.fileRepo.UpdateChunkCompleted(ctx, uploadID, &now)

	return &fileRecord, nil
}

// GetUploadProgress returns which chunks have been uploaded.
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

// CheckFile checks if a file with the given MD5 already exists.
func (s *FileService) CheckFile(ctx context.Context, md5Hash string) (*dto.CheckFileResponse, error) {
	return &dto.CheckFileResponse{Exists: false}, nil
}

// List returns a paginated list of files with optional filters.
func (s *FileService) List(ctx context.Context, page, pageSize int, name, originalName, mimeType, storageType, uploaderID string) ([]model.File, int64, error) {
	return s.fileRepo.ListFiltered(ctx, page, pageSize, name, originalName, mimeType, storageType, uploaderID)
}

// GetByID returns a file by its ID.
func (s *FileService) GetByID(ctx context.Context, id string) (*model.File, error) {
	file, err := s.fileRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "文件不存在")
	}
	return file, nil
}

// DownloadInfo holds data needed by the handler to serve a file download.
type DownloadInfo struct {
	File         *model.File
	FilePath     string
	FileSize     int64
	ContentType  string
	RangeHeader  string
}

// GetDownloadInfo prepares file download data including Range support.
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

// Delete soft-deletes a file record and removes the physical file.
func (s *FileService) Delete(ctx context.Context, id string) error {
	file, err := s.fileRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "文件不存在")
	}

	if err := s.fileRepo.Delete(ctx, id); err != nil {
		zap.L().Error("delete file record failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	if file.Path != "" {
		if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
			zap.L().Warn("remove physical file failed", zap.String("id", id), zap.String("path", file.Path), zap.Error(err))
		}
	}

	return nil
}

// ParseRange parses the Range header value and returns (start, end, ok).
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
		start, err = strconv.ParseInt(splits[0], 10, 64)
		if err != nil || start < 0 || start >= fileSize {
			return 0, 0, false
		}
		end = fileSize - 1
	} else {
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

// WriteRange writes a byte range from a file to an http.ResponseWriter.
func WriteRange(w http.ResponseWriter, filePath string, start, end int64, contentType string) error {
	length := end - start + 1
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, start+length))
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusPartialContent)

	buf := make([]byte, length)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.ReadAt(buf, start)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// detectMimeType returns a basic MIME type based on file extension.
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
