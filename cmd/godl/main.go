package main

import (
	"context"
	"download/internal/downloader"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/vbauerster/mpb/v8"
)

func main() {

	outputPath := flag.String("o", "", "Output file path")
	flag.Parse()

	if *outputPath == "" || flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: godl -o OUTPUT URL")
		flag.PrintDefaults()
		os.Exit(2)
	}

	progress := mpb.New(mpb.WithWidth(64))
	adapter := &mpbAdapter{progress: progress}

	err := downloader.Download(context.Background(), http.DefaultClient, flag.Arg(0), *outputPath, adapter)

	if err != nil {
		adapter.Abort()
	}

	progress.Wait()

	if err != nil {
		fmt.Fprintf(os.Stderr, "download: %v\n", err)
		os.Exit(1)
	}

}
