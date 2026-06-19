// Package log provides centralized logging initialization with zap and lumberjack.
// It supports splitting logs by function (access/app/error) with automatic rotation.
package log

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/niko-admin/niko-admin/internal/config"
)

// dailyRotateSyncer implements zapcore.WriteSyncer with date-based log rotation.
//
// It creates a new log file each day with a date-based filename pattern:
//
//	cfg.Path = "logs/app.log"  →  logs/app-2025-01-01.log
//	cfg.Path = "logs/error.log" → logs/error-2025-01-01.log
//
// Old files are cleaned up according to cfg.MaxAge and cfg.MaxBackups.
// When cfg.Compress is true, old files are gzip-compressed after rotation.
type dailyRotateSyncer struct {
	dir      string
	prefix   string // e.g. "app" from "logs/app.log"
	suffix   string // e.g. ".log"

	maxAge   time.Duration
	maxFiles int
	compress bool

	curDate string   // "2006-01-02"
	curFile *os.File
	mu      sync.Mutex
	stopCh  chan struct{}
}

// newDailyRotateSyncer creates a new dailyRotateSyncer from a LogFileConfig.
// It opens the current day's file immediately and starts a background goroutine
// to proactively rotate at midnight.
func newDailyRotateSyncer(cfg config.LogFileConfig) *dailyRotateSyncer {
	dir := filepath.Dir(cfg.Path)
	base := filepath.Base(cfg.Path) // e.g. "app.log"
	ext := filepath.Ext(base)       // ".log"
	prefix := strings.TrimSuffix(base, ext)

	if dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}

	w := &dailyRotateSyncer{
		dir:      dir,
		prefix:   prefix,
		suffix:   ext,
		maxAge:   time.Duration(cfg.MaxAge) * 24 * time.Hour,
		maxFiles: cfg.MaxBackups,
		compress: cfg.Compress,
		stopCh:   make(chan struct{}),
	}

	// Open current day's file and clean up stale archives
	// cleanupLocked is idempotent: only files matching prefix-YYYY-MM-DD.ext are affected
	now := time.Now()
	w.mu.Lock()
	w.cleanupLocked()
	if err := w.openFile(now); err != nil {
		w.mu.Unlock()
		_, _ = fmt.Fprintf(os.Stderr, "log: failed to open daily log file: %v\n", err)
		// curFile stays nil — Write() falls back to stderr
	} else {
		w.mu.Unlock()
	}

	// Background midnight rotation loop
	go w.rotateLoop()

	return w
}

// filenameFor returns the absolute log file path for a given time.
func (w *dailyRotateSyncer) filenameFor(t time.Time) string {
	return filepath.Join(w.dir, fmt.Sprintf("%s-%s%s", w.prefix, t.Format("2006-01-02"), w.suffix))
}

// openFile opens (or creates) the log file for the given time.
func (w *dailyRotateSyncer) openFile(now time.Time) error {
	date := now.Format("2006-01-02")
	fname := w.filenameFor(now)

	f, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", fname, err)
	}

	w.curDate = date
	w.curFile = f
	return nil
}

// Write implements io.Writer. It checks whether the date has changed and
// rotates to a new file if so, then writes to the current file.
func (w *dailyRotateSyncer) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.curFile == nil {
		// Fallback: write to stderr if no log file is available
		return os.Stderr.Write(p)
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	if date != w.curDate {
		// Date boundary crossed — rotate
		if err := w.rotateLocked(now); err != nil {
			return 0, err
		}
	}

	return w.curFile.Write(p)
}

// Sync implements zapcore.WriteSyncer.
func (w *dailyRotateSyncer) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.curFile != nil {
		return w.curFile.Sync()
	}
	return nil
}

// Close closes the current log file and stops the background rotation loop.
func (w *dailyRotateSyncer) Close() error {
	close(w.stopCh)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.curFile != nil {
		return w.curFile.Close()
	}
	return nil
}

// rotateLocked performs a date-boundary rotation. Must be called with w.mu held.
func (w *dailyRotateSyncer) rotateLocked(now time.Time) error {
	if w.curFile != nil {
		_ = w.curFile.Close()
		w.curFile = nil
	}

	// Compress the previous day's file if configured
	if w.compress {
		prevDate := w.curDate
		w.compressFileLocked(prevDate)
	}

	// Cleanup old files
	w.cleanupLocked()

	// Open new day's file
	return w.openFile(now)
}

// rotateLoop is a background goroutine that proactively rotates at midnight,
// so the first write of the new day doesn't block on file operations.
func (w *dailyRotateSyncer) rotateLoop() {
	for {
		now := time.Now()
		// Next midnight
		next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		timer := time.NewTimer(next.Sub(now))

		select {
		case <-timer.C:
			w.mu.Lock()
			// Check if Write hasn't already rotated us past this boundary
			today := time.Now()
			if today.Format("2006-01-02") != w.curDate {
				_ = w.rotateLocked(today)
			}
			w.mu.Unlock()

		case <-w.stopCh:
			timer.Stop()
			return
		}

		timer.Stop()
	}
}

// compressFileLocked gzip-compresses a daily log file.
// Renames <prefix>-<date>.log → <prefix>-<date>.log.gz.
// Silently skips missing/already-compressed files.
func (w *dailyRotateSyncer) compressFileLocked(date string) {
	src := filepath.Join(w.dir, fmt.Sprintf("%s-%s%s", w.prefix, date, w.suffix))
	dst := src + ".gz"

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return
	}
	if _, err := os.Stat(dst); err == nil {
		return // already compressed
	}

	// Read source
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}

	// Write compressed
	dstFile, err := os.Create(dst)
	if err != nil {
		return
	}
	defer dstFile.Close()

	gz := gzip.NewWriter(dstFile)
	if _, err := gz.Write(data); err != nil {
		_ = gz.Close()
		return
	}
	if err := gz.Close(); err != nil {
		return
	}

	_ = os.Remove(src)
}

// cleanupLocked removes old log files exceeding MaxAge or MaxBackups.
func (w *dailyRotateSyncer) cleanupLocked() {
	files := w.listLogFilesLocked()
	if len(files) <= 1 {
		return
	}

	// Sort by date ascending (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].date.Before(files[j].date)
	})

	var toRemove []logFileInfo
	cutoff := time.Now().Add(-w.maxAge)

	// 1. Remove files beyond MaxAge
	if w.maxAge > 0 {
		for _, f := range files {
			if f.date.Before(cutoff) {
				toRemove = append(toRemove, f)
			}
		}
	}

	// 2. If still over MaxBackups, remove oldest
	if w.maxFiles > 0 {
		removeSet := make(map[string]bool, len(toRemove))
		for _, f := range toRemove {
			removeSet[f.path] = true
		}

		remaining := 0
		for _, f := range files {
			if !removeSet[f.path] {
				remaining++
			}
		}

		if remaining > w.maxFiles {
			excess := remaining - w.maxFiles
			for _, f := range files {
				if removeSet[f.path] {
					continue
				}
				if excess <= 0 {
					break
				}
				toRemove = append(toRemove, f)
				removeSet[f.path] = true
				excess--
			}
		}
	}

	for _, f := range toRemove {
		_ = os.Remove(f.path)
		_ = os.Remove(f.path + ".gz")
	}
}

// logFileInfo holds metadata about a discovered log file.
type logFileInfo struct {
	path string
	date time.Time
}

// listLogFilesLocked scans the log directory for files matching <prefix>-YYYY-MM-DD<ext>.
func (w *dailyRotateSyncer) listLogFilesLocked() []logFileInfo {
	var files []logFileInfo

	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Match: <prefix>-YYYY-MM-DD<ext> or <prefix>-YYYY-MM-DD<ext>.gz
		var dateStr string
		switch {
		case strings.HasPrefix(name, w.prefix+"-") && strings.HasSuffix(name, w.suffix):
			// app-2025-01-01.log
			dateStr = strings.TrimPrefix(name, w.prefix+"-")
			dateStr = strings.TrimSuffix(dateStr, w.suffix)
		case strings.HasPrefix(name, w.prefix+"-") && strings.HasSuffix(name, w.suffix+".gz"):
			// app-2025-01-01.log.gz
			dateStr = strings.TrimPrefix(name, w.prefix+"-")
			dateStr = strings.TrimSuffix(dateStr, w.suffix+".gz")
		default:
			continue
		}

		// Parse date string
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		files = append(files, logFileInfo{
			path: filepath.Join(w.dir, name),
			date: t,
		})
	}

	return files
}
