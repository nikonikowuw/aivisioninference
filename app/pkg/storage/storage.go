// Package storage provides a unified interface for file storage backends
// including local filesystem, PostgreSQL Large Objects, and object storage (MinIO/S3).
package storage

import "io"

// Storage defines the interface for file storage backends.
type Storage interface {
	// Save stores data from reader and returns the storage path.
	Save(reader io.Reader, path string) (string, error)
	// Get returns a reader for the file at path.
	Get(path string) (io.ReadCloser, error)
	// Delete removes the file at path.
	Delete(path string) error
	// GetURL returns the public URL for the file.
	GetURL(path string) string
	// ReadAt reads len(p) bytes from the file starting at byte offset off.
	ReadAt(path string, p []byte, off int64) (n int, err error)
	// Size returns the file size in bytes.
	Size(path string) (int64, error)
}
