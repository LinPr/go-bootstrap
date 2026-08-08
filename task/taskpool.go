package task

import (
	"context"
	"sync"
)

// TaskPool collects tasks and executes them concurrently using a fixed-size worker pool.
type TaskPool[T any] struct {
	workers int
	tasks   []func(context.Context) T
	mu      *sync.Mutex
}

// NewTaskPool creates a TaskPool with the given number of worker goroutines.
// workers is clamped to 1 if <= 0.
func NewTaskPool[T any](workers int) *TaskPool[T] {
	if workers <= 0 {
		workers = 1
	}
	return &TaskPool[T]{workers: workers, mu: &sync.Mutex{}}
}

// Submit adds a task to the pool's task list.
func (p *TaskPool[T]) Submit(fn func(context.Context) T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tasks = append(p.tasks, fn)
}

// Run starts the worker goroutines, feeds all submitted tasks into the job queue,
// waits for completion, and returns results in submission order.
func (p *TaskPool[T]) Run(ctx context.Context) []T {

	results := make([]T, len(p.tasks))
	if len(p.tasks) == 0 {
		return results
	}

	type job struct {
		idx int
		fn  func(context.Context) T
	}

	// fill job channel upfront, then close so workers know when to stop
	jobCh := make(chan job, len(p.tasks))
	for i, fn := range p.tasks {
		jobCh <- job{idx: i, fn: fn}
	}
	close(jobCh)

	// 启动指定数量的 workers

	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case j, ok := <-jobCh:
					if !ok {
						return
					}
					results[j.idx] = j.fn(ctx)
				}
			}
		})
	}

	wg.Wait()
	return results
}
