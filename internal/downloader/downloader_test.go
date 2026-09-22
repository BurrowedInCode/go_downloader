package downloader

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type recordingReporter struct {
	total int64
	added int64
}

func (r *recordingReporter) SetTotal(total int64) {
	r.total = total
}

func (r *recordingReporter) Add(bytes int64) {
	r.added += bytes
}

type noopReporter struct{}

func (noopReporter) SetTotal(int64) {}

func (noopReporter) Add(int64) {}

func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloading"))
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "message.txt")

	reporter := &noopReporter{}

	err := Download(context.Background(), server.Client(), server.URL, outputPath, reporter)

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

	reporter := &noopReporter{}

	err := Download(context.Background(), server.Client(), server.URL, outputPath, reporter)

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

	reporter := &noopReporter{}

	err := Download(context.Background(), server.Client(), server.URL, outputPath, reporter)

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

	reporter := &noopReporter{}

	err := Download(context.Background(), server.Client(), server.URL, outputPath, reporter)

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

func TestDownloadSetsTotalContentLength(t *testing.T) {
	payload := []byte("download contents")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	reporter := &recordingReporter{}

	err := Download(context.Background(), server.Client(), server.URL, filepath.Join(t.TempDir(), "output.txt"), reporter)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	want := int64(len(payload))
	if reporter.total != want {
		t.Errorf("reported total = %d, want %d", reporter.total, want)
	}
}

func TestDownloadReportsWrittenBytes(t *testing.T) {
	payload := []byte("download contents")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	reporter := &recordingReporter{}
	outputPath := filepath.Join(t.TempDir(), "output.txt")

	err := Download(context.Background(), server.Client(), server.URL, outputPath, reporter)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	want := int64(len(payload))
	if reporter.added != want {
		t.Errorf("reported bytes = %d want %d", reporter.added, want)
	}
}

func TestProgressWriterReportsWrittenBytes(t *testing.T) {
	writer := &bytes.Buffer{}
	reporter := &recordingReporter{}
	payload := []byte("download contents")

	pr := &progressWriter{w: writer, reporter: reporter}

	n, err := pr.Write(payload)
	if err != nil {
		t.Fatalf("download progress failed to write: %v", err)
	}

	if n != len(payload) {
		t.Errorf("got %d bytes, want %d bytes", n, len(payload))
	}

	if reporter.added != int64(n) {
		t.Errorf("reporter add %d bytes but got %d bytes", reporter.added, int64(n))
	}

	if !bytes.Equal(writer.Bytes(), payload) {
		t.Errorf("written data = %q, want %q", writer.Bytes(), payload)
	}
}
