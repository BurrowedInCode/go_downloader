package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func Download(ctx context.Context, client *http.Client, rawURL, outputPath string) error {
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("output file already exists: %s", outputPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check output file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	partPath := outputPath + ".part"

	partFile, err := os.Create(partPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	_, copyErr := io.Copy(partFile, resp.Body)
	closeErr := partFile.Close()

	if copyErr != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("stream file: %w", copyErr)
	}

	if closeErr != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("close file: %w", closeErr)
	}

	if err := os.Rename(partPath, outputPath); err != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("rename file: %w", err)
	}

	return nil

}
