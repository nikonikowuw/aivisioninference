package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

func TestFileServiceInitUploadRejectsOversizedFile(t *testing.T) {
	svc := newTestFileService(t, FileOptions{MaxFileSizeBytes: 10, MaxChunkSizeBytes: 5})

	_, err := svc.InitUpload(context.Background(), dto.InitUploadRequest{
		FileName:    "big.txt",
		FileSize:    11,
		MD5:         fmt.Sprintf("%x", md5.Sum([]byte("big"))),
		TotalChunks: 1,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "文件大小超过限制")
}

func TestFileServiceInitUploadRejectsSVG(t *testing.T) {
	svc := newTestFileService(t, FileOptions{MaxFileSizeBytes: 1024, MaxChunkSizeBytes: 512})

	_, err := svc.InitUpload(context.Background(), dto.InitUploadRequest{
		FileName:    "xss.svg",
		FileSize:    10,
		MD5:         fmt.Sprintf("%x", md5.Sum([]byte("svg"))),
		TotalChunks: 1,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "不支持上传 SVG")
}

func TestFileServiceInitUploadRejectsMismatchedChunkCount(t *testing.T) {
	svc := newTestFileService(t, FileOptions{MaxFileSizeBytes: 1024, MaxChunkSizeBytes: 4})

	_, err := svc.InitUpload(context.Background(), dto.InitUploadRequest{
		FileName:    "small.txt",
		FileSize:    8,
		MD5:         fmt.Sprintf("%x", md5.Sum([]byte("12345678"))),
		TotalChunks: 1,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "分片数量与文件大小不匹配")
}

func TestFileServiceMaxChunkRequestBytesIncludesMultipartOverhead(t *testing.T) {
	svc := newTestFileService(t, FileOptions{MaxFileSizeBytes: 1024, MaxChunkSizeBytes: 4})

	require.Equal(t, int64(4+(1<<20)), svc.MaxChunkRequestBytes())
}

func TestFileServiceSaveChunkRejectsOversizedChunk(t *testing.T) {
	svc := newTestFileService(t, FileOptions{MaxFileSizeBytes: 1024, MaxChunkSizeBytes: 4})
	chunk, err := svc.InitUpload(context.Background(), dto.InitUploadRequest{
		FileName:    "small.txt",
		FileSize:    8,
		MD5:         fmt.Sprintf("%x", md5.Sum([]byte("12345678"))),
		TotalChunks: 2,
	})
	require.NoError(t, err)

	err = svc.SaveChunk(context.Background(), chunk.UploadID, 0, strings.NewReader("12345"))

	require.Error(t, err)
	require.Contains(t, err.Error(), "分片大小超过限制")
}

func TestFileServiceCheckFileFindsExistingMD5(t *testing.T) {
	svc := newTestFileService(t, FileOptions{MaxFileSizeBytes: 1024, MaxChunkSizeBytes: 512})
	repo := svc.fileRepo
	file := &model.File{Name: "stored.txt", OriginalName: "stored.txt", Path: "uploads/stored.txt", MimeType: "text/plain", Size: 6, StorageType: "local", MD5: "abc123"}
	require.NoError(t, repo.Create(context.Background(), file))

	resp, err := svc.CheckFile(context.Background(), "abc123")

	require.NoError(t, err)
	require.True(t, resp.Exists)
	require.Equal(t, file.ID, resp.FileID)
}

func TestFileServiceCheckFileReturnsNotExistsWhenMD5Missing(t *testing.T) {
	svc := newTestFileService(t, FileOptions{MaxFileSizeBytes: 1024, MaxChunkSizeBytes: 512})

	resp, err := svc.CheckFile(context.Background(), "missing")

	require.NoError(t, err)
	require.False(t, resp.Exists)
	require.Empty(t, resp.FileID)
}

func TestFileServiceCheckFileReturnsErrorOnDatabaseFailure(t *testing.T) {
	svc, db := newTestFileServiceWithDB(t, FileOptions{MaxFileSizeBytes: 1024, MaxChunkSizeBytes: 512})
	require.NoError(t, db.Exec("DROP TABLE files").Error)

	resp, err := svc.CheckFile(context.Background(), "abc123")

	require.Error(t, err)
	require.Nil(t, resp)
}

func newTestFileService(t *testing.T, opts FileOptions) *FileService {
	t.Helper()
	svc, _ := newTestFileServiceWithDB(t, opts)
	return svc
}

func newTestFileServiceWithDB(t *testing.T, opts FileOptions) (*FileService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE files (
		id text primary key,
		created_at datetime,
		updated_at datetime,
		deleted_at datetime,
		created_by text,
		updated_by text,
		name text not null,
		original_name text not null,
		path text,
		loid integer,
		mime_type text,
		size integer,
		md5 text,
		storage_type text not null
	)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_files_md5 ON files (md5)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE file_chunks (
		id text primary key,
		created_at datetime,
		updated_at datetime,
		deleted_at datetime,
		created_by text,
		updated_by text,
		upload_id text unique not null,
		file_name text not null,
		file_size integer,
		md5 text not null,
		total_chunks integer,
		status text not null,
		storage_type text not null,
		uploaded_chunks text,
		expires_at datetime,
		completed_at datetime
	)`).Error)

	baseDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(baseDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "tmp", "uploads"), 0755))

	return NewFileService(repository.NewFileRepository(db), opts), db
}
