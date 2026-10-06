package main

import (
	"context"
	"download/internal/downloader"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vbauerster/mpb/v8"
)

func run() int {
	outputPath := flag.String("o", "", "Output file path")
	flag.Parse()

	if *outputPath == "" || flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: godl -o OUTPUT URL")
		flag.PrintDefaults()
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	progress := mpb.New(mpb.WithWidth(64))
	adapter := &mpbAdapter{progress: progress}

	err := downloader.Download(ctx, http.DefaultClient, flag.Arg(0), *outputPath, adapter)

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
	return 0
}

func main() {
	os.Exit(run())
}
