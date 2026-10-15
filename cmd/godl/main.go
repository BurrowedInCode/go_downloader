package main

import (
	"context"
	"download/internal/downloader"
	"errors"
	"flag"
	"fmt"
	"github.com/vbauerster/mpb/v8"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
)

func run() int {
	outputDirectory := flag.String("o", "", "Output directory")
	concurrency := flag.Int("j", 4, "Maximum concurrent downloads")
	flag.Parse()

	urls := flag.Args()

	if *outputDirectory == "" || len(urls) == 0 {
		fmt.Fprintln(os.Stderr, "usage: godl -o DIRECTORY [-j N] URL...")
		flag.PrintDefaults()
		return 2
	}

	if *concurrency < 1 {
		fmt.Fprintln(os.Stderr, "-j must be at least 1")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		outputPath := filepath.Join(*outputDirectory, fileName)

		progress := mpb.New(mpb.WithWidth(64))
		adapter := &mpbAdapter{progress: progress}

		err = downloader.Download(ctx, http.DefaultClient, rawURL, outputPath, adapter)

		if err != nil {
			adapter.Abort()
		} else {
			adapter.Complete()
		}

		progress.Wait()

		if errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "download cancelled\n")
			return 130
		} else if err != nil {

			fmt.Fprintf(os.Stderr, "download: %v\n", err)
			return 1
		}
	}
	return 0
}

func main() {
	os.Exit(run())
}
