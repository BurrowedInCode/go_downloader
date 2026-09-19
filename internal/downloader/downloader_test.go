package downloader

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloading"))
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "message.txt")

	err := Download(context.Background(), server.Client(), server.URL, outputPath)

	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	want := "downloading"
	if string(got) != want {
		t.Fatalf("output = %q, want %q", got, want)
	}

	if _, err := os.Stat(outputPath + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial file still exists: %v", err)
	}
}

func TestDownloadReturnsErrorFor404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "message.txt")

	err := Download(context.Background(), server.Client(), server.URL, outputPath)

	if err == nil {
		t.Fatal("Download() error = nil, want HTTP status error")
	}

	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("completed file exists or stat failed: %v", err)
	}

	if _, err := os.Stat(outputPath + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("partial file exists or stat failed: %v", err)
	}
}

func TestDownloadDoesNotOverwriteExistingFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("replacement"))
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "original.txt")

	if err := os.WriteFile(outputPath, []byte("original"), 0o644); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	err := Download(context.Background(), server.Client(), server.URL, outputPath)

	if err == nil {
		t.Fatal("Download() error = nil want existing-file error")
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}

	const want = "original"

	if string(got) != want {
		t.Errorf("output = %q, want %q", got, want)
	}

	if _, err := os.Stat(outputPath + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("partial file exists or stat failed: %v", err)
	}
}

func TestDownloadRemovesPartialFileWhenResponseIsIncomplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("partial"))
	}))

	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "original.txt")

	err := Download(context.Background(), server.Client(), server.URL, outputPath)

	if err == nil {
		t.Fatal("Download() error = nil want incomplete-response error")
	}

	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("partial file exists or stat failed: %v", err)

	}
	if _, err := os.Stat(outputPath + ".part"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("completed file exists or stat failed: %v", err)
	}
}
