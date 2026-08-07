package task

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTaskPool(t *testing.T) {
	t.Run("positive workers", func(t *testing.T) {
		pool := NewTaskPool[int](5)
		if pool.workers != 5 {
			t.Errorf("expected 5 workers, got %d", pool.workers)
		}
	})

	t.Run("zero workers defaults to 1", func(t *testing.T) {
		pool := NewTaskPool[int](0)
		if pool.workers != 1 {
			t.Errorf("expected 1 worker, got %d", pool.workers)
		}
	})

	t.Run("negative workers defaults to 1", func(t *testing.T) {
		pool := NewTaskPool[int](-5)
		if pool.workers != 1 {
			t.Errorf("expected 1 worker, got %d", pool.workers)
		}
	})
}

func TestTaskPool_Submit(t *testing.T) {
	pool := NewTaskPool[int](2)

	pool.Submit(func(ctx context.Context) (int, error) {
		return 1, nil
	})
	pool.Submit(func(ctx context.Context) (int, error) {
		return 2, nil
	})

	if len(pool.tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(pool.tasks))
	}
}

func TestTaskPool_Run_EmptyPool(t *testing.T) {
	pool := NewTaskPool[int](2)
	ctx := context.Background()

	results := pool.Run(ctx)

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestTaskPool_Run_SingleTask(t *testing.T) {
	pool := NewTaskPool[int](2)
	ctx := context.Background()

	pool.Submit(func(ctx context.Context) (int, error) {
		return 42, nil
	})

	results := pool.Run(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Value != 42 {
		t.Errorf("expected value 42, got %d", results[0].Value)
	}

	if results[0].Err != nil {
		t.Errorf("expected no error, got %v", results[0].Err)
	}
}

func TestTaskPool_Run_MultipleTasks(t *testing.T) {
	pool := NewTaskPool[int](3)
	ctx := context.Background()

	// Submit tasks that return their index
	for i := 0; i < 10; i++ {
		val := i
		pool.Submit(func(ctx context.Context) (int, error) {
			time.Sleep(10 * time.Millisecond)
			return val * 2, nil
		})
	}

	results := pool.Run(ctx)

	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}

	// Verify results are in submission order
	for i, result := range results {
		expected := i * 2
		if result.Value != expected {
			t.Errorf("result[%d]: expected %d, got %d", i, expected, result.Value)
		}
		if result.Err != nil {
			t.Errorf("result[%d]: unexpected error %v", i, result.Err)
		}
	}
}

func TestTaskPool_Run_WithErrors(t *testing.T) {
	pool := NewTaskPool[string](2)
	ctx := context.Background()

	testErr := errors.New("test error")

	pool.Submit(func(ctx context.Context) (string, error) {
		return "success", nil
	})

	pool.Submit(func(ctx context.Context) (string, error) {
		return "", testErr
	})

	pool.Submit(func(ctx context.Context) (string, error) {
		return "another success", nil
	})

	results := pool.Run(ctx)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Value != "success" || results[0].Err != nil {
		t.Error("first task should succeed")
	}

	if results[1].Err != testErr {
		t.Errorf("second task should have error, got %v", results[1].Err)
	}

	if results[2].Value != "another success" || results[2].Err != nil {
		t.Error("third task should succeed")
	}
}

func TestTaskPool_Run_ContextCancellation(t *testing.T) {
	pool := NewTaskPool[int](2)
	ctx, cancel := context.WithCancel(context.Background())

	var completedCount atomic.Int32

	// Submit tasks that would take a while
	for i := 0; i < 5; i++ {
		pool.Submit(func(ctx context.Context) (int, error) {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				completedCount.Add(1)
				return 1, nil
			}
		})
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	results := pool.Run(ctx)

	// Some tasks should have been cancelled
	cancelledCount := 0
	for _, result := range results {
		if errors.Is(result.Err, context.Canceled) {
			cancelledCount++
		}
	}

	if cancelledCount == 0 {
		t.Error("expected some tasks to be cancelled")
	}

	t.Logf("completed: %d, cancelled: %d", completedCount.Load(), cancelledCount)
}

func TestTaskPool_Run_ConcurrentExecution(t *testing.T) {
	pool := NewTaskPool[int](5)
	ctx := context.Background()

	var counter atomic.Int32
	var maxConcurrent atomic.Int32

	// Submit tasks that track concurrency
	for i := 0; i < 20; i++ {
		pool.Submit(func(ctx context.Context) (int, error) {
			current := counter.Add(1)
			
			// Track max concurrent execution
			for {
				max := maxConcurrent.Load()
				if current <= max || maxConcurrent.CompareAndSwap(max, current) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			counter.Add(-1)
			return int(current), nil
		})
	}

	results := pool.Run(ctx)

	if len(results) != 20 {
		t.Errorf("expected 20 results, got %d", len(results))
	}

	maxReached := maxConcurrent.Load()
	t.Logf("max concurrent executions: %d", maxReached)

	if maxReached > 5 {
		t.Errorf("max concurrent should not exceed 5, got %d", maxReached)
	}

	if maxReached < 2 {
		t.Error("expected at least 2 concurrent executions")
	}
}

func TestTaskPool_Run_DifferentTypes(t *testing.T) {
	t.Run("string type", func(t *testing.T) {
		pool := NewTaskPool[string](2)
		ctx := context.Background()

		pool.Submit(func(ctx context.Context) (string, error) {
			return "hello", nil
		})

		pool.Submit(func(ctx context.Context) (string, error) {
			return "world", nil
		})

		results := pool.Run(ctx)

		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}

		if results[0].Value != "hello" {
			t.Errorf("expected 'hello', got '%s'", results[0].Value)
		}

		if results[1].Value != "world" {
			t.Errorf("expected 'world', got '%s'", results[1].Value)
		}
	})

	t.Run("struct type", func(t *testing.T) {
		type Result struct {
			ID   int
			Name string
		}

		pool := NewTaskPool[Result](2)
		ctx := context.Background()

		pool.Submit(func(ctx context.Context) (Result, error) {
			return Result{ID: 1, Name: "test"}, nil
		})

		results := pool.Run(ctx)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}

		if results[0].Value.ID != 1 || results[0].Value.Name != "test" {
			t.Errorf("unexpected result: %+v", results[0].Value)
		}
	})
}

func TestTaskPool_Run_MultipleRuns(t *testing.T) {
	pool := NewTaskPool[int](2)
	ctx := context.Background()

	// First run
	pool.Submit(func(ctx context.Context) (int, error) {
		return 1, nil
	})

	results1 := pool.Run(ctx)
	if len(results1) != 1 || results1[0].Value != 1 {
		t.Error("first run failed")
	}

	// Second run with same pool (tasks should be from first run)
	results2 := pool.Run(ctx)
	if len(results2) != 1 || results2[0].Value != 1 {
		t.Error("second run should reuse tasks")
	}
}

func BenchmarkTaskPool(b *testing.B) {
	ctx := context.Background()

	b.Run("small_tasks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pool := NewTaskPool[int](4)
			for j := 0; j < 10; j++ {
				val := j
				pool.Submit(func(ctx context.Context) (int, error) {
					return val, nil
				})
			}
			pool.Run(ctx)
		}
	})

	b.Run("many_tasks", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pool := NewTaskPool[int](8)
			for j := 0; j < 100; j++ {
				val := j
				pool.Submit(func(ctx context.Context) (int, error) {
					return val * 2, nil
				})
			}
			pool.Run(ctx)
		}
	})
}
