# godl — Concurrent CLI Download Manager

`godl` is a Go command-line tool for downloading files from direct HTTP(S) URLs. It streams responses to disk, limits concurrent downloads, displays progress, and supports cancellation.

## Current Features

- Download multiple URLs using a bounded worker pool.
- Configure the maximum number of simultaneous downloads.
- Stream files to disk without buffering entire responses in memory.
- Display progress bars, or a spinner when the response length is unknown.
- Propagate Ctrl+C and SIGTERM cancellation to HTTP requests and workers.
- Write to `.part` files and rename them after successful streaming and file close.
- Reject output files that already exist at the initial check.
- Report individual download failures without stopping unrelated downloads.

Retries, resumable downloads, per-host limits, and parallel range downloads are **not implemented yet**.

## Requirements

- Go 1.26.5 or newer, as declared in `go.mod`.
- An existing, writable output directory.
- Direct HTTP or HTTPS URLs with usable filenames in their paths.

## Build

From the repository root:

```bash
go build -o godl ./cmd/godl
```

Alternatively, run the CLI without building a separate executable:

```bash
go run ./cmd/godl -o downloads 'https://example.com/data.csv'
```

Create the output directory first. The URLs in this document are placeholders; replace them with real download links.

## Usage

```text
godl -o DIRECTORY [-j N] URL...
```

| Flag | Description | Default |
| --- | --- | --- |
| `-o` | Output directory; required and must already exist | None |
| `-j` | Maximum concurrent downloads; must be at least 1 | `4` |

Place flags before the URLs. Quote URLs, especially those containing query parameters such as `?` and `&`.

### Download one file

```bash
mkdir -p downloads
./godl -o downloads 'https://example.com/data.csv'
```

The output filename comes from the URL path, so this saves to `downloads/data.csv`.

### Download several files

```bash
./godl -o downloads -j 2 \
  'https://example.com/data.csv' \
  'https://example.com/assets.zip' \
  'https://example.com/report.pdf'
```

At most two downloads run at once. Other files wait for an available worker.

### Download a list of URLs with Bash

Put one URL per line in `urls.txt`, with no blank lines or comments:

```text
https://example.com/data.csv
https://example.com/assets.zip
https://example.com/report.pdf
```

Then run in Bash:

```bash
mkdir -p downloads
mapfile -t urls < urls.txt
./godl -o downloads -j 4 "${urls[@]}"
```

URL-list parsing is provided by the shell in this example, not by a built-in CLI flag.

### Cancel downloads

Press **Ctrl+C** to cancel. Cancellation propagates to active requests, and the manager waits for its workers to exit. Interrupted downloads restart from the beginning on the next run; they cannot currently resume.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | All downloads succeeded |
| `1` | At least one download failed |
| `2` | Invalid arguments or flag usage |
| `130` | The operation was canceled through Ctrl+C or SIGTERM |

## Practical Uses

- Fetch public datasets for analysis or testing.
- Download release archives and development tools.
- Collect project assets from known direct URLs.
- Make repeated download tasks reproducible in shell scripts.

The tool downloads HTTP responses. It does not crawl websites, discover download links, or extract files from HTML pages.

## File Handling and Limitations

- **Filename selection:** Filenames come from the supplied URL paths, not `Content-Disposition` headers. URLs without usable filenames, including paths ending in `/`, are rejected.
- **Naming collisions:** Collision detection between jobs is not implemented. Ensure every output path and its corresponding `.part` path are distinct from those of other jobs. For example, do not download two URLs ending in `data.csv`, or schedule both `archive.zip` and `archive.zip.part` in the same directory.
- **Existing files:** A completed output file detected before the request causes an error. This check is not an atomic no-overwrite guarantee; avoid concurrent processes writing to the same destination.
- **Partial files:** Downloads use `<filename>.part`. An existing `.part` file is truncated when writing starts; it is not resumed. Cleanup is attempted after streaming, file-close, or rename failures, but cleanup errors are currently ignored. Abrupt process termination may also leave partial files behind.
- **Retries:** HTTP and network failures are reported without automatic retry.
- **Timeouts:** There is no CLI-configurable request timeout or overall download deadline. Ctrl+C can cancel stalled downloads.
- **Concurrency:** The limit applies globally to download jobs. There are no separate per-host limits or parallel chunks within a single file.

## Architecture

```text
CLI: arguments, signals, progress, exit status
                  |
                  v
Manager: bounded workers, job scheduling, results
                  |
                  v
Downloader: HTTP request -> streamed .part file -> rename
```

The manager owns worker lifecycle and job/result channels. The downloader accepts a context and a reusable HTTP client. Progress is reported through a small interface, with terminal rendering handled by the CLI's `mpb` adapter.

### Repository Layout

```text
cmd/godl/
  main.go              CLI arguments and application lifecycle
  progress.go          Terminal progress adapter
internal/downloader/
  downloader.go        HTTP streaming and file handling
  downloader_test.go   Downloader and progress-accounting tests
internal/manager/
  manager.go           Worker pool and result collection
  manager_test.go      Concurrency and cancellation tests
```

The download and scheduling logic uses Go's standard library. Terminal progress rendering uses [`mpb`](https://github.com/vbauerster/mpb).

## Development and Verification

Run from the repository root:

```bash
go test ./...
go test -race ./...
go vet ./...
go build -o godl ./cmd/godl
```

Format changed Go files with `gofmt` before committing. Tests use controlled local HTTP handlers and synchronization to exercise streaming, progress accounting, failure cleanup, concurrency limits, result ordering, and cancellation.

## Roadmap

Planned incremental improvements:

1. Validate output and `.part` path collisions before starting downloads.
2. Add bounded retries with failure classification and cancellable exponential backoff.
3. Support resume using HTTP Range requests and validated `206 Partial Content` responses.
4. Add cancellation-aware per-host concurrency limits.
5. Optionally explore parallel range downloads after ordinary downloads and resume behavior are well tested.

The priority is a smaller, well-tested downloader whose concurrency and failure behavior can be explained directly from the source.
