package manager

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

func TestRunPopulatesResults(t *testing.T) {
	job1 := DownloadJob{
		OutputPath: filepath.Join(t.TempDir(), "file.txt"),
		URL:        "https://example.com/file.txt"}
	job2 := DownloadJob{
		OutputPath: filepath.Join(t.TempDir(), "file1.txt"),
		URL:        "https://example.com/file1.txt"}

	jobs := []DownloadJob{job1, job2}

	downloadError := errors.New("error downloading")

	downloadFunc := func(ctx context.Context, job DownloadJob) error {
		if job.URL == jobs[1].URL {
			return downloadError
		}
		return nil
	}

	manager := NewManager(1, downloadFunc)

	results := manager.Run(context.Background(), jobs)

	if len(results) != len(jobs) {
		t.Fatalf("got %d results, want %d", len(results), len(jobs))
	}

	if results[0].Job != jobs[0] {
		t.Errorf("got job %+v, want %+v", results[0].Job, jobs[0])
	}

	if results[0].Err != nil {
		t.Errorf("got %v, want nil", results[0].Err)
	}

	if results[1].Job != jobs[1] {
		t.Errorf("got job %+v, want %+v", results[1].Job, jobs[1])
	}

	if !errors.Is(results[1].Err, downloadError) {
		t.Errorf("got error %v, want %v", results[1].Err, downloadError)
	}
}

func TestRunDownloadsConcurrently(t *testing.T) {
	started := make(chan DownloadJob, 2)
	release := make(chan struct{})

	job1 := DownloadJob{
		OutputPath: filepath.Join(t.TempDir(), "file.txt"),
		URL:        "https://example.com/file.txt"}
	job2 := DownloadJob{
		OutputPath: filepath.Join(t.TempDir(), "file1.txt"),
		URL:        "https://example.com/file1.txt"}

	jobs := []DownloadJob{job1, job2}

	downloadFunc := func(ctx context.Context, job DownloadJob) error {
		started <- job

		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	manager := NewManager(2, downloadFunc)

	go func() {
		manager.Run(ctx, jobs)
		close(done)
	}()

	t.Cleanup(func() {
		cancel()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("Run did not stop after cancellation")
		}
	})

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for both downloads to start")
		}
	}

	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not finish after releasing downloads")
	}
}

func TestRunReturnsResultsInOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan DownloadJob, 2)
		release := make(chan struct{})

		job1 := DownloadJob{
			OutputPath: filepath.Join(t.TempDir(), "file.txt"),
			URL:        "https://example.com/file.txt"}
		job2 := DownloadJob{
			OutputPath: filepath.Join(t.TempDir(), "file1.txt"),
			URL:        "https://example.com/file1.txt"}

		jobs := []DownloadJob{job1, job2}

		downloadFunc := func(ctx context.Context, job DownloadJob) error {
			started <- job

			if job == jobs[0] {
				select {
				case <-release:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		}

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})

		manager := NewManager(2, downloadFunc)

		var results []Result

		go func() {
			results = manager.Run(ctx, jobs)
			close(done)
		}()

		t.Cleanup(func() {
			cancel()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Error("Run did not stop after cancellation")
			}

		})

		synctest.Wait()

		if len(started) != len(jobs) {
			t.Fatalf("got %d, want %d", len(started), len(jobs))
		}

		select {
		case <-done:
			t.Fatal("Unexpectedly ended goroutine")
		default:
		}
		close(release)

		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatal("Run has not finished")
		}

		if len(results) != len(jobs) {
			t.Fatalf("got %d results, want %d", len(results), len(jobs))
		}

		for i, job := range jobs {
			if results[i].Job != job {
				t.Errorf("result %d: got job %+v, want %+v", i, results[i].Job, job)
			}
		}
	})
}
