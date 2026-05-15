package dto

// InitUploadRequest is the request body for initializing a chunked upload.
type InitUploadRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required,gt=0"`
	MD5         string `json:"md5" binding:"required"`
	TotalChunks int    `json:"total_chunks" binding:"required,gt=0"`
	StorageType string `json:"storage_type"`
}

// CheckFileRequest is the request body for checking file existence by MD5.
type CheckFileRequest struct {
	MD5 string `json:"md5" binding:"required"`
}

// CheckFileResponse is the response for file existence check.
type CheckFileResponse struct {
	Exists bool `json:"exists"`
	FileID string `json:"file_id,omitempty"`
}

// UploadProgressResponse is the response for upload progress query.
type UploadProgressResponse struct {
	UploadID       string `json:"upload_id"`
	UploadedChunks []int  `json:"uploaded_chunks"`
	TotalChunks    int    `json:"total_chunks"`
}
