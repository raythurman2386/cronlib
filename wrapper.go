package cronlib

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobWrapper decorates a job execution function.
// It allows injecting custom logic before and after the job runs (middleware).
type JobWrapper func(func(context.Context) error) func(context.Context) error

// Chain combines multiple job wrappers into a single wrapper.
// Wrappers are executed in the order they are passed.
// Example: Chain(W1, W2) results in W1(W2(Job)).
func Chain(wrappers ...JobWrapper) JobWrapper {
	return func(next func(context.Context) error) func(context.Context) error {
		for i := len(wrappers) - 1; i >= 0; i-- {
			next = wrappers[i](next)
		}
		return next
	}
}

// Recover creates a wrapper that catches panics and returns them as errors.
func Recover() JobWrapper {
	return func(next func(context.Context) error) func(context.Context) error {
		return func(ctx context.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v", r)
				}
			}()
			return next(ctx)
		}
	}
}

// logWrapper returns a wrapper that logs job execution details to the store.
func (c *Cron) logWrapper(jobID string) JobWrapper {
	return func(next func(context.Context) error) func(context.Context) error {
		return func(ctx context.Context) error {
			c.mu.RLock()
			store := c.store
			c.mu.RUnlock()

			if store == nil {
				return next(ctx)
			}
			start := time.Now()

			err := next(ctx)

			end := time.Now()
			success := err == nil
			msg := ""
			if err != nil {
				msg = err.Error()
			}

			_ = store.LogExecution(jobID, start, end, success, msg)
			_ = store.SetLastRun(jobID, start)

			return err
		}
	}
}

// lockWrapper returns a wrapper that handles distributed locking.
func (c *Cron) lockWrapper(jobID string) JobWrapper {
	return func(next func(context.Context) error) func(context.Context) error {
		return func(ctx context.Context) error {
			c.mu.RLock()
			lock := c.lock
			c.mu.RUnlock()

			if lock == nil {
				return next(ctx)
			}

			// 1 minute TTL hardcoded for now, as per original
			locked, err := lock.Lock(ctx, "cron:"+jobID, time.Minute)
			if err != nil || !locked {
				// Failed to acquire lock, skip execution
				return nil // Return nil or specific error? skipping is "success" in terms of "no crash"
			}
			defer func() { _ = lock.Unlock(ctx, "cron:"+jobID) }()

			return next(ctx)
		}
	}
}

// DelayIfStillRunning ensures that job executions are sequential.
// If a job is triggered while a previous instance is running, it waits.
func DelayIfStillRunning() JobWrapper {
	var mu sync.Mutex
	return func(next func(context.Context) error) func(context.Context) error {
		return func(ctx context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			return next(ctx)
		}
	}
}

// SkipIfStillRunning skips execution if the previous instance is still running.
// This is an alternative to the OverlapForbid policy, implemented as a wrapper.
func SkipIfStillRunning() JobWrapper {
	var mu sync.Mutex
	var running bool
	return func(next func(context.Context) error) func(context.Context) error {
		return func(ctx context.Context) error {
			mu.Lock()
			if running {
				mu.Unlock()
				return nil
			}
			running = true
			mu.Unlock()

			defer func() {
				mu.Lock()
				running = false
				mu.Unlock()
			}()

			return next(ctx)
		}
	}
}
