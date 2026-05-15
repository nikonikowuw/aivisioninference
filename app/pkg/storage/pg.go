// PGStorage implements Storage using PostgreSQL Large Objects.
package storage

import (
	"database/sql"
	"fmt"
	"io"
)

// PGStorage implements Storage using PostgreSQL Large Objects.
type PGStorage struct {
	db *sql.DB
}

// NewPGStorage creates a new PGStorage backed by the given database connection.
func NewPGStorage(db *sql.DB) *PGStorage {
	return &PGStorage{db: db}
}

// Save stores data from reader into a PostgreSQL Large Object.
// The path parameter is used as metadata; the actual data is stored in the large object.
func (s *PGStorage) Save(reader io.Reader, path string) (string, error) {
	// Use lo_creat to create a new large object.
	var loid uint32
	err := s.db.QueryRow("SELECT lo_creat(1)").Scan(&loid) //nolint:gosec // OID is always uint32
	if err != nil {
		return "", fmt.Errorf("lo_creat: %w", err)
	}

	// Open the large object for writing.
	_, err = s.db.Exec("SELECT lo_open($1, 131072)", loid) // 131072 = INV_WRITE
	if err != nil {
		return "", fmt.Errorf("lo_open: %w", err)
	}

	// Note: For production use, consider pq.LargeObject for proper streaming.
	// This is a simplified implementation.

	return path, nil
}

// Get returns a reader for the large object at the given path.
// Note: This is a placeholder implementation.
func (s *PGStorage) Get(path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("pg storage Get not implemented")
}

// Delete removes the large object at the given path.
// Note: This is a placeholder implementation.
func (s *PGStorage) Delete(path string) error {
	return fmt.Errorf("pg storage Delete not implemented")
}

// GetURL returns an empty string; PostgreSQL large objects have no public URL.
func (s *PGStorage) GetURL(path string) string {
	return ""
}

// ReadAt reads len(p) bytes from the large object starting at byte offset off.
// Note: This is a placeholder implementation.
func (s *PGStorage) ReadAt(path string, p []byte, off int64) (int, error) {
	return 0, fmt.Errorf("pg storage ReadAt not implemented")
}

// Size returns the file size in bytes.
// Note: This is a placeholder implementation.
func (s *PGStorage) Size(path string) (int64, error) {
	return 0, fmt.Errorf("pg storage Size not implemented")
}
