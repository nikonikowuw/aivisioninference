package service

import (
	"archive/tar"
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
)

func TestBuildAlgoDownloadURL(t *testing.T) {
	tests := []struct {
		name      string
		publicURL string
		token     string
		want      string
	}{
		{
			name:      "absolute uploads public url",
			publicURL: "http://localhost:8080/uploads",
			token:     "abc",
			want:      "http://localhost:8080/api/v1/internal/algo/download?token=abc",
		},
		{
			name:      "relative uploads public url",
			publicURL: "/uploads",
			token:     "abc",
			want:      "/api/v1/internal/algo/download?token=abc",
		},
		{
			name:      "token escaped",
			publicURL: "http://localhost:8080/uploads/",
			token:     "a+b c",
			want:      "http://localhost:8080/api/v1/internal/algo/download?token=a%2Bb+c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildAlgoDownloadURL(tt.publicURL, tt.token); got != tt.want {
				t.Fatalf("buildAlgoDownloadURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// createTestZip writes a zip to dir with the given entries (name → content).
// Each entry is written directly so directories must have explicit entries or
// be implicit from file paths.
func createTestZip(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	zipFile, err := os.CreateTemp(dir, "test-*.zip")
	if err != nil {
		t.Fatalf("create temp zip: %v", err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip writer close: %v", err)
	}
	return zipFile.Name()
}

// readTarEntries returns a set of entry names in the tar at path.
func readTarEntries(t *testing.T, path string) map[string]bool {
	t.Helper()
	r, err := os.Open(path)
	if err != nil {
		t.Fatalf("open tar: %v", err)
	}
	defer r.Close()

	entries := make(map[string]bool)
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		// Normalize: strip trailing slash for consistent lookup
		name := strings.TrimSuffix(hdr.Name, "/")
		entries[name] = hdr.FileInfo().IsDir()
	}
	return entries
}

func TestRepackZipToTar(t *testing.T) {
	t.Run("flat layout with top-level directory", func(t *testing.T) {
		dir := t.TempDir()
		zipPath := createTestZip(t, dir, map[string]string{
			"face_recognition_1.0.0/algo_meta.yaml":           "algorithm_name: test_algo\nversion: 1.0.0\n",
			"face_recognition_1.0.0/test_algo.so":            "fake-so-binary",
			"face_recognition_1.0.0/models/face_model.onnx":  "fake-onnx-data",
		})

		tarPath := dir + "/out.tar"
		meta, size, md5sum, err := RepackZipToTar(zipPath, tarPath)
		if err != nil {
			t.Fatalf("RepackZipToTar failed: %v", err)
		}
		if meta.AlgorithmName != "test_algo" {
			t.Fatalf("unexpected algorithm name: %s", meta.AlgorithmName)
		}
		if size <= 0 {
			t.Fatal("expected positive size")
		}
		if md5sum == "" {
			t.Fatal("expected non-empty md5")
		}

		entries := readTarEntries(t, tarPath)

		// .so must be at root (no top-level directory prefix)
		if _, ok := entries["test_algo.so"]; !ok {
			t.Fatalf("expected test_algo.so at root, got: %v", keys(entries))
		}
		// algo_meta.yaml must be at root
		if _, ok := entries["algo_meta.yaml"]; !ok {
			t.Fatalf("expected algo_meta.yaml at root, got: %v", keys(entries))
		}
		// models/... must be preserved
		if _, ok := entries["models/face_model.onnx"]; !ok {
			t.Fatalf("expected models/face_model.onnx, got: %v", keys(entries))
		}
		// Top-level directory must NOT appear
		if _, ok := entries["face_recognition_1.0.0"]; ok {
			t.Fatal("top-level directory should be stripped")
		}
	})

	t.Run("nested models preserved under stripped root", func(t *testing.T) {
		dir := t.TempDir()
		zipPath := createTestZip(t, dir, map[string]string{
			"algo_pkg_2.1.0/algo_meta.yaml":                "algorithm_name: test_algo\nversion: 2.1.0\n",
			"algo_pkg_2.1.0/nikoniko_detector.so":          "fake-so",
			"algo_pkg_2.1.0/models/face_1.onnx":            "data1",
			"algo_pkg_2.1.0/models/face_2.onnx":            "data2",
			"algo_pkg_2.1.0/models/sub/deep.onnx":          "deep",
		})

		tarPath := dir + "/out.tar"
		_, _, _, err := RepackZipToTar(zipPath, tarPath)
		if err != nil {
			t.Fatalf("RepackZipToTar failed: %v", err)
		}

		entries := readTarEntries(t, tarPath)

		// .so at root
		if _, ok := entries["nikoniko_detector.so"]; !ok {
			t.Fatal("expected nikoniko_detector.so at root")
		}
		// models/... preserved
		if _, ok := entries["models/face_1.onnx"]; !ok {
			t.Fatal("expected models/face_1.onnx")
		}
		if _, ok := entries["models/face_2.onnx"]; !ok {
			t.Fatal("expected models/face_2.onnx")
		}
		if _, ok := entries["models/sub/deep.onnx"]; !ok {
			t.Fatal("expected models/sub/deep.onnx")
		}
		// top-level dir stripped
		if _, ok := entries["algo_pkg_2.1.0"]; ok {
			t.Fatal("top-level directory should be stripped")
		}
	})

	t.Run("already flat package preserved", func(t *testing.T) {
		dir := t.TempDir()
		zipPath := createTestZip(t, dir, map[string]string{
			"algo_meta.yaml":                "algorithm_name: flat_algo\nversion: 1.0.0\n",
			"nikoniko_detector.so":          "fake-so",
			"models/face_model.onnx":        "fake-onnx",
		})

		tarPath := dir + "/out.tar"
		_, _, _, err := RepackZipToTar(zipPath, tarPath)
		if err != nil {
			t.Fatalf("RepackZipToTar failed: %v", err)
		}

		entries := readTarEntries(t, tarPath)

		if _, ok := entries["nikoniko_detector.so"]; !ok {
			t.Fatal("expected nikoniko_detector.so at root")
		}
		if _, ok := entries["algo_meta.yaml"]; !ok {
			t.Fatal("expected algo_meta.yaml at root")
		}
		if _, ok := entries["models/face_model.onnx"]; !ok {
			t.Fatal("expected models/face_model.onnx")
		}
	})

	t.Run("zip slip path rejected", func(t *testing.T) {
		dir := t.TempDir()
		zipPath := createTestZip(t, dir, map[string]string{
			"../evil.so":                                     "evil",
			"algo_meta.yaml":                                 "algorithm_name: test\nversion: 1.0.0\n",
		})

		tarPath := dir + "/out.tar"
		_, _, _, err := RepackZipToTar(zipPath, tarPath)
		if err == nil {
			t.Fatal("expected zip slip error")
		}
		if !strings.Contains(err.Error(), "zip slip") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("absolute path rejected", func(t *testing.T) {
		dir := t.TempDir()
		zipPath := createTestZip(t, dir, map[string]string{
			"/etc/passwd":                                    "evil",
			"algo_meta.yaml":                                 "algorithm_name: test\nversion: 1.0.0\n",
		})

		tarPath := dir + "/out.tar"
		_, _, _, err := RepackZipToTar(zipPath, tarPath)
		if err == nil {
			t.Fatal("expected zip slip error")
		}
		if !strings.Contains(err.Error(), "zip slip") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("dup slash path rejected", func(t *testing.T) {
		dir := t.TempDir()
		zipPath := createTestZip(t, dir, map[string]string{
			"safe/../../evil.so":                              "evil",
			"algo_meta.yaml":                                 "algorithm_name: test\nversion: 1.0.0\n",
		})

		tarPath := dir + "/out.tar"
		_, _, _, err := RepackZipToTar(zipPath, tarPath)
		if err == nil {
			t.Fatal("expected zip slip error")
		}
		if !strings.Contains(err.Error(), "zip slip") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("missing meta rejected", func(t *testing.T) {
		dir := t.TempDir()
		zipPath := createTestZip(t, dir, map[string]string{
			"test_algo.so": "fake-so",
		})

		tarPath := dir + "/out.tar"
		_, _, _, err := RepackZipToTar(zipPath, tarPath)
		if err == nil || !strings.Contains(err.Error(), "algo_meta.yaml not found") {
			t.Fatalf("expected meta-not-found error, got: %v", err)
		}
	})

	t.Run("missing so rejected", func(t *testing.T) {
		dir := t.TempDir()
		zipPath := createTestZip(t, dir, map[string]string{
			"algo_meta.yaml": "algorithm_name: test\nversion: 1.0.0\n",
		})

		tarPath := dir + "/out.tar"
		_, _, _, err := RepackZipToTar(zipPath, tarPath)
		if err == nil || !strings.Contains(err.Error(), ".so file not found") {
			t.Fatalf("expected so-not-found error, got: %v", err)
		}
	})
}

func keys(m map[string]bool) []string {
	var s []string
	for k := range m {
		s = append(s, k)
	}
	return s
}
