package cronlib

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestDelayIfStillRunning(t *testing.T) {
	var events []string
	var mu sync.Mutex

	record := func(msg string) {
		mu.Lock()
		events = append(events, msg)
		mu.Unlock()
	}

	wrapper := DelayIfStillRunning()

	// Create a job function that sleeps
	jobFunc := func(id int) func(context.Context) error {
		return func(ctx context.Context) error {
			record(fmt.Sprintf("start %d", id))
			time.Sleep(100 * time.Millisecond)
			record(fmt.Sprintf("end %d", id))
			return nil
		}
	}

	// Wrap the jobs
	// Note: DelayIfStillRunning returns a wrapper closure with a shared mutex.
	// We must use the SAME wrapper instance for both calls if we want them to exclude each other.
	// But `AddJob` creates a NEW chain for each job?
	// Wait, `AddJob` calls `opts.Wrappers`. If I pass the SAME wrapper instance in opts, it works?
	// `DelayIfStillRunning()` returns a `JobWrapper`.
	// `JobWrapper` is `func(next) next`.
	// The closure `var mu sync.Mutex` is created when `DelayIfStillRunning()` is called.
	// If I pass `wrapper` (the result of `DelayIfStillRunning()`) to `AddJob`, then yes, it shares the mutex.
	
	// However, `AddJob` applies the wrapper chain: `Chain(wrappers...)(baseCmd)`.
	// `Chain` calls `wrappers[i](next)`.
	// If `wrapper` captures `mu`, then every time `wrapper(next)` is called, it returns a function that uses `mu`.
	// Yes.
	
	// BUT, `AddJob` is called ONCE per job definition.
	// The `finalCmd` is stored in `job.Cmd`.
	// So `job.Cmd` is the one holding the mutex.
	// If the job runs multiple times, `job.Cmd` is called multiple times.
	// Does `job.Cmd` share state across executions?
	// Yes, `job.Cmd` is a closure returned by `Chain`.
	// The `Chain` builds the function once.
	// So subsequent calls to `job.Cmd` share the closure environment.
	// So `DelayIfStillRunning` works correctly for a SINGLE job instance running multiple times.
	
	// Testing logic:
	// We need to simulate calling the wrapped function concurrently.
	
	wrappedJob := wrapper(jobFunc(1))
	
	var wg sync.WaitGroup
	wg.Add(2)
	
	go func() {
		defer wg.Done()
		wrappedJob(context.Background())
	}()
	
	time.Sleep(20 * time.Millisecond) // Ensure first starts
	
	go func() {
		defer wg.Done()
		wrappedJob(context.Background())
	}()
	
	wg.Wait()
	
	// Verify order: start 1, end 1, start 1, end 1 (or 1 then 2 if we passed different IDs but same wrapper)
	// Here we called `wrappedJob` twice.
	
	if len(events) != 4 {
		t.Fatalf("Expected 4 events, got %d", len(events))
	}
	
	// Expected: start, end, start, end.
	// If parallel: start, start, end, end (or mixed).
	// Because of sleep, if parallel, we'd see start, start.
	
	if events[0] != "start 1" {
		t.Errorf("Event 0 expected start 1, got %s", events[0])
	}
	if events[1] != "end 1" {
		t.Errorf("Event 1 expected end 1, got %s (Parallel execution detected?)", events[1])
	}
	if events[2] != "start 1" {
		t.Errorf("Event 2 expected start 1, got %s", events[2])
	}
	if events[3] != "end 1" {
		t.Errorf("Event 3 expected end 1, got %s", events[3])
	}
}

func TestRecoverWrapper(t *testing.T) {
	wrapper := Recover()
	
	panickyJob := func(ctx context.Context) error {
		panic("oops")
	}
	
	wrapped := wrapper(panickyJob)
	
	err := wrapped(context.Background())
	if err == nil {
		t.Error("Expected error from panic, got nil")
	}
	
	if err.Error() != "panic: oops" {
		t.Errorf("Expected 'panic: oops', got '%v'", err)
	}
}
