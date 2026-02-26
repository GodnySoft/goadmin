package core

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRunsJobs(t *testing.T) {
	var count int32
	sched := NewScheduler(10 * time.Millisecond)
	sched.Add(func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	sched.Start(ctx)
	if c := atomic.LoadInt32(&count); c == 0 {
		t.Fatalf("expected jobs to run, got %d", c)
	}
}
