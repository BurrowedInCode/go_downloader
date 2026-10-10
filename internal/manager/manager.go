package manager

import (
	"context"
	"sync"
)

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

type indexedJob struct {
	index int
	job   DownloadJob
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

	if m.workers <= 0 {
		m.workers = 2
	}

	results := make([]Result, len(jobs))

	jobsChannel := make(chan indexedJob)
	resultsChannel := make(chan indexedResult, len(jobs))

	go func() {
		defer close(jobsChannel)
		for i, job := range jobs {
			select {
			case jobsChannel <- indexedJob{index: i, job: job}:
			case <-ctx.Done():
				return
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(m.workers)

	for i := 0; i < m.workers; i++ {
		go func() {
			defer wg.Done()
			for item := range jobsChannel {
				if ctx.Err() != nil {
					return
				}
				err := m.downloadFunc(ctx, item.job)
				result := Result{Job: item.job, Err: err}
				resultsChannel <- indexedResult{index: item.index, result: result}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsChannel)
	}()

	received := make([]bool, len(jobs))

	for result := range resultsChannel {
		results[result.index] = result.result
		received[result.index] = true
	}

	if err := ctx.Err(); err != nil {
		for i, wasReceived := range received {
			if !wasReceived {
				results[i] = Result{Job: jobs[i], Err: err}
			}
		}
	}

	return results
}
