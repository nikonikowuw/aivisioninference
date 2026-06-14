package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LocalStorage implements Storage using the local filesystem.
type LocalStorage struct {
	uploadDir string
	publicURL string
}

// NewLocalStorage creates a new LocalStorage with the given upload directory and public URL base.
// It ensures the upload directory exists.
func NewLocalStorage(uploadDir, publicURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStorage{uploadDir: uploadDir, publicURL: publicURL}, nil
}

// Save stores data from reader to the local filesystem at the given path.
// It creates intermediate directories as needed.
func (s *LocalStorage) Save(reader io.Reader, path string) (string, error) {
	fullPath := filepath.Join(s.uploadDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("create parent dir: %w", err)
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return path, nil
}

// Get returns a reader for the file at the given path.
func (s *LocalStorage) Get(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.uploadDir, path))
}

// Delete removes the file at the given path.
func (s *LocalStorage) Delete(path string) error {
	return os.Remove(filepath.Join(s.uploadDir, path))
}

// GetURL returns the public URL for the file.
func (s *LocalStorage) GetURL(path string) string {
	return s.publicURL + "/" + path
}

// GetPresignedURL returns the public URL for the file (local storage doesn't support expiration).
func (s *LocalStorage) GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	return s.GetURL(path), nil
}

// ReadAt reads len(p) bytes from the file starting at byte offset off.
func (s *LocalStorage) ReadAt(path string, p []byte, off int64) (int, error) {
	f, err := os.Open(filepath.Join(s.uploadDir, path))
	if err != nil {
		return 0, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	n, err := f.ReadAt(p, off)
	if err != nil {
		return n, fmt.Errorf("read at: %w", err)
	}
	return n, nil
}

// Size returns the file size in bytes.
func (s *LocalStorage) Size(path string) (int64, error) {
	info, err := os.Stat(filepath.Join(s.uploadDir, path))
	if err != nil {
		return 0, fmt.Errorf("stat file: %w", err)
	}
	return info.Size(), nil
}
