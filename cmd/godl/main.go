package main

import (
	"context"
	"download/internal/downloader"
	"download/internal/manager"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vbauerster/mpb/v8"
)

// TODO: Check for naming collisions between jobs' output and .part paths before starting downloads.
func run() int {
	outputDirectory := flag.String("o", "", "Output directory")
	workers := flag.Int("j", 4, "Maximum concurrent downloads")
	flag.Parse()

	urls := flag.Args()

	if *outputDirectory == "" || len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "usage: godl -o DIRECTORY [-j N] URL...")
		flag.PrintDefaults()
		return 2
	}

	if *workers < 1 {
		fmt.Fprintln(os.Stderr, "-j must be at least 1")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var jobs []manager.DownloadJob

	for _, rawURL := range urls {
		u, err := url.Parse(rawURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid URL: ", err)
			return 2
		}

		if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
			fmt.Fprintf(os.Stderr, "expected an HTTP(S) URL with a host %q\n", rawURL)
			return 2
		}

		fileName := path.Base(u.Path)

		switch fileName {
		case "", ".", "..", "/":
			fmt.Fprintf(os.Stderr, "URL has no usable filename: %q\n", rawURL)
			return 2
		}

		if strings.HasSuffix(u.Path, "/") {
			fmt.Fprintf(os.Stderr, "URL path ends in a slash: %q\n", rawURL)
			return 2
		}

		outputPath := filepath.Join(*outputDirectory, fileName)

		jobs = append(jobs, manager.DownloadJob{OutputPath: outputPath, URL: rawURL})
	}

	progress := mpb.New(mpb.WithWidth(64))
	mgr := manager.NewManager(*workers, func(ctx context.Context, dj manager.DownloadJob) error {
		adapter := &mpbAdapter{progress: progress}

		err := downloader.Download(ctx, http.DefaultClient, dj.URL, dj.OutputPath, adapter)
		if err != nil {
			adapter.Abort()
		} else {
			adapter.Complete()
		}
		return err
	})

	results := mgr.Run(ctx, jobs)
	progress.Wait()

	exitCode := 0
	for _, result := range results {
		if result.Err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", result.Job.URL, result.Err)
			exitCode = 1
		}
	}

	if ctx.Err() != nil {
		return 130
	}

	return exitCode
}

func main() {
	os.Exit(run())
}
