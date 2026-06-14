package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// OSSStorage implements Storage using MinIO/S3-compatible object storage.
type OSSStorage struct {
	client *minio.Client
	bucket string
}

// NewOSSStorage creates a new OSSStorage connected to the given MinIO/S3 endpoint.
// It verifies the bucket exists and creates it if necessary.
func NewOSSStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*OSSStorage, error) {
	cred := credentials.NewStaticV4(accessKey, secretKey, "")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  cred,
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	// Set bucket policy to empty (private) to prevent anonymous access
	if err := client.SetBucketPolicy(ctx, bucket, ""); err != nil {
		return nil, fmt.Errorf("set bucket policy to private: %w", err)
	}

	return &OSSStorage{client: client, bucket: bucket}, nil
}

// Save stores data from reader into the object storage at the given path.
func (s *OSSStorage) Save(reader io.Reader, path string) (string, error) {
	_, err := s.client.PutObject(context.Background(), s.bucket, path, reader, -1, minio.PutObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	return path, nil
}

// Get returns a reader for the object at the given path.
func (s *OSSStorage) Get(path string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(context.Background(), s.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return obj, nil
}

// Delete removes the object at the given path.
func (s *OSSStorage) Delete(path string) error {
	return s.client.RemoveObject(context.Background(), s.bucket, path, minio.RemoveObjectOptions{})
}

// GetURL returns the bucket-relative path for the file.
func (s *OSSStorage) GetURL(path string) string {
	return fmt.Sprintf("/%s/%s", s.bucket, path)
}

// GetPresignedURL returns a time-limited public URL for the file.
func (s *OSSStorage) GetPresignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, path, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("presign object: %w", err)
	}
	return u.String(), nil
}

// ReadAt reads len(p) bytes from the object starting at byte offset off.
func (s *OSSStorage) ReadAt(path string, p []byte, off int64) (int, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(off, off+int64(len(p))-1); err != nil {
		return 0, fmt.Errorf("set range: %w", err)
	}
	obj, err := s.client.GetObject(context.Background(), s.bucket, path, opts)
	if err != nil {
		return 0, fmt.Errorf("get object range: %w", err)
	}
	defer obj.Close()
	return obj.Read(p)
}

// Size returns the object size in bytes.
func (s *OSSStorage) Size(path string) (int64, error) {
	info, err := s.client.StatObject(context.Background(), s.bucket, path, minio.StatObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("stat object: %w", err)
	}
	return info.Size, nil
}
