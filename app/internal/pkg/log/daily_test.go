package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/niko-admin/niko-admin/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// today returns the current date string used in log filenames.
func today() string {
	return time.Now().Format("2006-01-02")
}

func TestDailyRotateSyncer_FilenamePattern(t *testing.T) {
	dir := t.TempDir()

	cfg := config.LogFileConfig{
		Enabled:   true,
		Path:      filepath.Join(dir, "test-app.log"),
		TimeBased: true,
		MaxAge:    7,
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	// The current file should follow <prefix>-YYYY-MM-DD.log pattern
	expected := filepath.Join(dir, fmt.Sprintf("test-app-%s.log", today()))
	_, err := os.Stat(expected)
	require.NoError(t, err, "daily log file should exist with date-based name")
}

func TestDailyRotateSyncer_WriteAndRead(t *testing.T) {
	dir := t.TempDir()
	cfg := config.LogFileConfig{
		Enabled:   true,
		Path:      filepath.Join(dir, "test-app.log"),
		TimeBased: true,
		MaxAge:    7,
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	_, err := w.Write([]byte("hello\n"))
	require.NoError(t, err)
	_, err = w.Write([]byte("world\n"))
	require.NoError(t, err)

	// Read back
	data, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("test-app-%s.log", today())))
	require.NoError(t, err)
	assert.Equal(t, "hello\nworld\n", string(data))
}

func TestDailyRotateSyncer_Sync(t *testing.T) {
	dir := t.TempDir()
	cfg := config.LogFileConfig{
		Enabled:   true,
		Path:      filepath.Join(dir, "test-sync.log"),
		TimeBased: true,
		MaxAge:    7,
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	// Sync should not error
	err := w.Sync()
	assert.NoError(t, err)
}

func TestDailyRotateSyncer_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	cfg := config.LogFileConfig{
		Enabled:   true,
		Path:      filepath.Join(dir, "test-conc.log"),
		TimeBased: true,
		MaxAge:    7,
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := w.Write([]byte(fmt.Sprintf("line %d\n", n)))
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Verify all lines were written (file should exist and contain all lines)
	fpath := filepath.Join(dir, fmt.Sprintf("test-conc-%s.log", today()))
	data, err := os.ReadFile(fpath)
	require.NoError(t, err)
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	assert.Equal(t, 10, lines, "all 10 concurrent writes should appear in the log file")
}

func TestDailyRotateSyncer_CleanupByMaxAge(t *testing.T) {
	dir := t.TempDir()

	// Create old log files that should be cleaned up
	oldFiles := []string{
		filepath.Join(dir, "test-app-2024-01-01.log"),
		filepath.Join(dir, "test-app-2024-01-02.log"),
		filepath.Join(dir, "test-app-2024-01-03.log"),
	}
	for _, f := range oldFiles {
		require.NoError(t, os.WriteFile(f, []byte("old data\n"), 0644))
	}

	// Create a current file (should be preserved)
	currentFile := filepath.Join(dir, fmt.Sprintf("test-app-%s.log", today()))
	require.NoError(t, os.WriteFile(currentFile, []byte("current data\n"), 0644))

	cfg := config.LogFileConfig{
		Enabled:   true,
		Path:      filepath.Join(dir, "test-app.log"),
		TimeBased: true,
		MaxAge:    1, // keep only 1 day
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	// Trigger cleanup via rotate (create a fake next-day file by calling openFile with a future date)
	// Actually, we can directly test the cleanupLocked method by calling it through rotate
	// Better approach: just verify file existence after Init cleaned up
	// The issue is that cleanup is called during Write when date changes, or during rotateLocked
	// For this test, let's just verify the dir state after construction

	// Write something to trigger cleanup (since curDate matches today, no rotation needed)
	_, err := w.Write([]byte("new data\n"))
	require.NoError(t, err)

	// Old files should have been cleaned up
	for _, f := range oldFiles {
		_, err := os.Stat(f)
		assert.True(t, os.IsNotExist(err), "old file %s should be removed by MaxAge cleanup", f)
	}

	// Current file should still exist
	_, err = os.Stat(currentFile)
	assert.NoError(t, err, "current file should still exist")
}

func TestDailyRotateSyncer_CleanupByMaxBackups(t *testing.T) {
	dir := t.TempDir()

	// Create 10 old log files with sequential dates
	for i := 1; i <= 10; i++ {
		fname := filepath.Join(dir, fmt.Sprintf("test-app-2024-01-%02d.log", i))
		require.NoError(t, os.WriteFile(fname, []byte("old data\n"), 0644))
	}

	// Create current file (same name as constructor will open)
	currentFile := filepath.Join(dir, fmt.Sprintf("test-app-%s.log", today()))
	require.NoError(t, os.WriteFile(currentFile, []byte("current data\n"), 0644))

	// Also create a future file (should be newer than all)
	futureFile := filepath.Join(dir, "test-app-2099-12-31.log")
	require.NoError(t, os.WriteFile(futureFile, []byte("future data\n"), 0644))

	cfg := config.LogFileConfig{
		Enabled:    true,
		Path:       filepath.Join(dir, "test-app.log"),
		TimeBased:  true,
		MaxAge:     0,    // disable age-based cleanup
		MaxBackups: 6,   // keep only 6 most recent
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	// Write to trigger initial cleanup (constructor also cleans up)
	_, err := w.Write([]byte("new data\n"))
	require.NoError(t, err)

	// After cleanup with MaxBackups=6:
	// Total 12 files: 10 old (2024-01-01..10) + today + future (2099-12-31)
	// Sorted oldest-first: 01-01,01-02,...,01-10,today,2099-12-31
	// Keep 6 most recent: 01-07..01-10 (4) + today + future = 6
	// Remove 6 oldest: 01-01..01-06

	// Verify files that should be removed (6 oldest)
	for i := 1; i <= 6; i++ {
		fname := filepath.Join(dir, fmt.Sprintf("test-app-2024-01-%02d.log", i))
		_, err := os.Stat(fname)
		assert.True(t, os.IsNotExist(err), "old file 2024-01-%02d should be removed", i)
	}

	// Verify files that should be preserved (4 most recent old)
	for i := 7; i <= 10; i++ {
		fname := filepath.Join(dir, fmt.Sprintf("test-app-2024-01-%02d.log", i))
		_, err := os.Stat(fname)
		assert.NoError(t, err, "recent file 2024-01-%02d should be preserved", i)
	}

	// Future and today should be preserved
	_, err = os.Stat(currentFile)
	assert.NoError(t, err, "current file should be preserved")
	_, err = os.Stat(futureFile)
	assert.NoError(t, err, "future file should be preserved")

	// Total should be 6 (4 recent + today + future)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 6, "should keep exactly 6 log files after MaxBackups=6 cleanup")
}

func TestDailyRotateSyncer_CompressOldFile(t *testing.T) {
	dir := t.TempDir()

	// Create a log file that will be compressed
	oldFile := filepath.Join(dir, "test-app-2024-06-01.log")
	err := os.WriteFile(oldFile, []byte("this is old data that should be compressed\n"), 0644)
	require.NoError(t, err)

	cfg := config.LogFileConfig{
		Enabled:   true,
		Path:      filepath.Join(dir, "test-app.log"),
		TimeBased: true,
		Compress:  true,
		MaxAge:    0, // disable age-based cleanup so the old file isn't removed
		MaxBackups: 100,
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	// Test compressFileLocked directly (same package, unexported is accessible)
	w.mu.Lock()
	w.compressFileLocked("2024-06-01")
	w.mu.Unlock()

	// Verify compression
	compressedPath := oldFile + ".gz"
	_, err = os.Stat(compressedPath)
	assert.NoError(t, err, "old file should be gzip compressed")

	// The raw file should no longer exist
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err), "raw file should be removed after compression")
}

func TestDailyRotateSyncer_DisabledFileLogging(t *testing.T) {
	// For disabled config, newFileWriter returns io.Discard without calling newDailyRotateSyncer.
	// Directly constructing newDailyRotateSyncer with Enabled:false and empty Path is not
	// a valid real-world path (Path is always set when TimeBased=true).
	// Instead verify that newFileWriter handles disabled correctly:
	w := newFileWriter(config.LogFileConfig{Enabled: false})
	_, err := w.Write([]byte("should not appear\n"))
	assert.NoError(t, err, "write should not error even with disabled logging")
	_ = w.Sync()

	// Also verify non-empty path but disabled returns io.Discard
	w2 := newFileWriter(config.LogFileConfig{Enabled: false, Path: "/tmp/fake.log", TimeBased: true})
	_, err = w2.Write([]byte("should not appear\n"))
	assert.NoError(t, err)
	_ = w2.Sync()
}

func TestDailyRotateSyncer_FilenameFor(t *testing.T) {
	dir := t.TempDir()
	cfg := config.LogFileConfig{
		Enabled:   true,
		Path:      filepath.Join(dir, "myapp.log"),
		TimeBased: true,
		MaxAge:    7,
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	// Verify filename format
	t1 := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	fname := w.filenameFor(t1)
	assert.Equal(t, filepath.Join(dir, "myapp-2025-06-01.log"), fname)
}

func TestDailyRotateSyncer_ListLogFiles(t *testing.T) {
	dir := t.TempDir()

	// Create files with various dates
	files := []string{
		filepath.Join(dir, "test-list-2025-01-01.log"),
		filepath.Join(dir, "test-list-2025-01-02.log"),
		filepath.Join(dir, "test-list-2025-01-03.log.gz"), // compressed
		filepath.Join(dir, "unrelated-file.txt"),            // should be ignored
		filepath.Join(dir, "test-list-no-date.log"),          // no date pattern, should be ignored
	}
	for _, f := range files {
		require.NoError(t, os.WriteFile(f, []byte("data\n"), 0644))
	}

	cfg := config.LogFileConfig{
		Enabled:   true,
		Path:      filepath.Join(dir, "test-list.log"),
		TimeBased: true,
		MaxAge:    0, // disable age-based cleanup so test files are preserved
	}
	w := newDailyRotateSyncer(cfg)
	defer w.Close()

	results := w.listLogFilesLocked()
	// 3 pre-created + 1 opened by newDailyRotateSyncer (today's date)
	require.Len(t, results, 4, "should find 3 pre-created + 1 current = 4 log files")

	// Verify the current day's file is included
	currentDate := time.Now().Format("2006-01-02")
	foundCurrent := false
	for _, r := range results {
		if r.date.Format("2006-01-02") == currentDate {
			foundCurrent = true
			break
		}
	}
	assert.True(t, foundCurrent, "current day's file should be discovered")

	// Verify all pre-created dates are present (plus current date)
	dateMap := make(map[string]bool)
	for _, r := range results {
		dateMap[r.date.Format("2006-01-02")] = true
	}
	assert.True(t, dateMap["2025-01-01"], "2025-01-01 should be found")
	assert.True(t, dateMap["2025-01-02"], "2025-01-02 should be found")
	assert.True(t, dateMap["2025-01-03"], "2025-01-03 should be found")
	assert.True(t, dateMap[time.Now().Format("2006-01-02")], "current date should be found")
}
