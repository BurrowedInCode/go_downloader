package manager

import "context"

type DownloadJob struct {
	OutputPath string
	URL        string
}

type Result struct {
	Job DownloadJob
	Err error
}

type Manager struct {
	workers      int
	downloadFunc func(context.Context, DownloadJob) error
}

type indexedResult struct {
	index  int
	result Result
}

func NewManager(workers int, downloadFunc func(context.Context, DownloadJob) error) *Manager {
	return &Manager{
		workers:      workers,
		downloadFunc: downloadFunc,
	}
}

func (m *Manager) Run(ctx context.Context, jobs []DownloadJob) []Result {
	results := make([]Result, len(jobs))
	resultsChannel := make(chan indexedResult, len(jobs))

	for i, job := range jobs {
		go func() {
			err := m.downloadFunc(ctx, job)
			result := &Result{Job: job, Err: err}
			resultsChannel <- indexedResult{index: i, result: *result}
		}()
	}

	for range len(jobs) {
		result := <-resultsChannel
		results[result.index] = result.result
	}

	return results
}
