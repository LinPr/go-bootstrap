package task

import (
	"context"
	"errors"
	"testing"
)

func TestTaskPool_Run_Int(t *testing.T) {
	pool := NewTaskPool[int](3)

	for i := 1; i <= 5; i++ {
		pool.Submit(func(ctx context.Context) int {
			return i * 2
		})
	}

	results := pool.Run(t.Context())
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	for i, got := range results {
		expected := (i + 1) * 2
		if got != expected {
			t.Fatalf("result[%d]: expected %d, got %d", i, expected, got)
		}
	}
}

func TestTaskPool_Run_ResultWithError(t *testing.T) {
	type ResultWithErr struct {
		Value int
		Err   error
	}

	expectedErr := errors.New("task failed")
	pool := NewTaskPool[ResultWithErr](2)

	pool.Submit(func(ctx context.Context) ResultWithErr {
		return ResultWithErr{Value: 10}
	})
	pool.Submit(func(ctx context.Context) ResultWithErr {
		return ResultWithErr{Err: expectedErr}
	})

	results := pool.Run(t.Context())
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Value != 10 || results[0].Err != nil {
		t.Fatalf("unexpected first result: %+v", results[0])
	}

	if !errors.Is(results[1].Err, expectedErr) {
		t.Fatalf("expected second error %v, got %v", expectedErr, results[1].Err)
	}
}
