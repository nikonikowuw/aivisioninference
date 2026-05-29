package dto

import (
	"time"

	"github.com/niko-admin/niko-admin/internal/pkg/scopes"
)

// InitUploadRequest is the request body for initializing a chunked upload.
type InitUploadRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required,gt=0"`
	MD5         string `json:"md5" binding:"required,len=32,hexadecimal"`
	TotalChunks int    `json:"total_chunks" binding:"required,gt=0"`
	StorageType string `json:"storage_type"`
}

// CheckFileRequest is the request body for checking file existence by MD5.
type CheckFileRequest struct {
	MD5 string `json:"md5" binding:"required,len=32,hexadecimal"`
}

// CheckFileResponse is the response for file existence check.
type CheckFileResponse struct {
	Exists bool   `json:"exists"`
	FileID string `json:"file_id,omitempty"`
}

// UploadProgressResponse is the response for upload progress query.
type UploadProgressResponse struct {
	UploadID       string `json:"upload_id"`
	UploadedChunks []int  `json:"uploaded_chunks"`
	TotalChunks    int    `json:"total_chunks"`
}

// FileListRequest is the request for listing files with filters.
type FileListRequest struct {
	PageRequest
	Keyword     string `form:"keyword"`
	MimeType    string `form:"mime_type"`
	StorageType string `form:"storage_type"`
	// UploaderID filters by upload creator. Maps to created_by via BaseModel.
	UploaderID string     `form:"uploader_id"`
	StartTime  string     `form:"start_time"`
	EndTime    string     `form:"end_time"`
	FromTime   *time.Time `form:"-" json:"-"`
	ToTime     *time.Time `form:"-" json:"-"`
}

func (r *FileListRequest) FilterScopes() []scopes.Scope {
	var sc []scopes.Scope
	if r.Keyword != "" {
		sc = append(sc, scopes.MultiLike([]string{"name", "original_name"}, r.Keyword))
	}
	if r.MimeType != "" {
		sc = append(sc, scopes.Eq("mime_type", r.MimeType))
	}
	if r.StorageType != "" {
		sc = append(sc, scopes.Eq("storage_type", r.StorageType))
	}
	if r.UploaderID != "" {
		sc = append(sc, scopes.Eq("created_by", r.UploaderID))
	}
	if r.FromTime != nil || r.ToTime != nil {
		sc = append(sc, scopes.TimeRange("created_at", r.FromTime, r.ToTime))
	}
	return sc
}
