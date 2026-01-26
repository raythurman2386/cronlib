package cronlib

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOverlapForbid(t *testing.T) {
	c := NewCron()
	c.Start()
	defer c.Stop()

	var wg sync.WaitGroup
	wg.Add(2)

	runCount := 0
	var mu sync.Mutex

	// Job runs 1.5s
	// Schedule: every 1s
	// T0: start (runs 1.5s)
	// T1: skip (running)
	// T2: start (prev finished at 1.5)

	_, err := c.AddJobWithOptions("* * * * * *", func(ctx context.Context) {
		mu.Lock()
		runCount++
		current := runCount
		mu.Unlock()

		t.Logf("Job %d started at %v", current, time.Now())

		select {
		case <-time.After(1500 * time.Millisecond):
			// Finished naturally
		case <-ctx.Done():
			t.Error("Job context cancelled in Forbid mode")
		}

		t.Logf("Job %d finished at %v", current, time.Now())
		if current <= 2 {
			wg.Done()
		}
	}, JobOptions{Overlap: OverlapForbid})

	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	// Wait for 2 runs (approx 4 seconds)
	// If it allowed overlap, it would run at 0, 1, 2, 3...
	// With forbid: 0 runs. 1 skips. 2 runs.

	// We wait for 2 completions.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("2 jobs finished")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for jobs")
	}

	mu.Lock()
	count := runCount
	mu.Unlock()

	// In 5 seconds (T0..T4), expected runs: T0, T2, T4.
	// We waited for 2 runs.
	if count > 3 {
		t.Errorf("Too many runs: %d, expected roughly 2-3", count)
	}
}

func TestOverlapReplace(t *testing.T) {
	c := NewCron()
	c.Start()
	defer c.Stop()

	var wg sync.WaitGroup
	wg.Add(2) // Wait for 2 cancellations to happen

	var cancelledCount int32

	// Job runs 2s. Schedule every 1s.
	// T0: Start.
	// T1: Trigger. T0 running. Cancel T0. Start T1.

	_, err := c.AddJobWithOptions("* * * * * *", func(ctx context.Context) {
		select {
		case <-time.After(2 * time.Second):
			// Finished naturally (should not happen for T0)
		case <-ctx.Done():
			// Cancelled
			// Only count first 2 cancellations
			if atomic.AddInt32(&cancelledCount, 1) <= 2 {
				t.Log("Job cancelled")
				wg.Done()
			}
		}
	}, JobOptions{Overlap: OverlapReplace})

	if err != nil {
		t.Fatal(err)
	}

	// We simply wait for a few cancellations
	// T0 starts. T1 cancels T0. T1 starts. T2 cancels T1.

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("Got cancellations")
	case <-time.After(4 * time.Second):
		t.Fatal("Timeout waiting for cancellations")
	}
}

func TestOverlapAllow(t *testing.T) {
	c := NewCron()
	c.Start()
	defer c.Stop()

	var wg sync.WaitGroup
	// We want to see multiple running at once.
	// Run 3 jobs, each takes 3s. Interval 1s.
	// T0: Start J1.
	// T1: Start J2. (J1 still running)
	// T2: Start J3. (J1, J2 running)

	wg.Add(3)

	active := 0
	maxActive := 0
	var mu sync.Mutex
	var runCount int32

	_, err := c.AddJobWithOptions("* * * * * *", func(ctx context.Context) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		count := atomic.AddInt32(&runCount, 1)

		defer func() {
			mu.Lock()
			active--
			mu.Unlock()
			if count <= 3 {
				wg.Done()
			}
		}()

		time.Sleep(2500 * time.Millisecond)
	}, JobOptions{Overlap: OverlapAllow}) // Default

	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout")
	}

	if maxActive < 2 {
		t.Errorf("Expected overlap (maxActive >= 2), got %d", maxActive)
	}
}
