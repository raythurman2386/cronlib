package cronlib

import (
	"context"
	"fmt"
	"sync"
)

// Chain combines multiple job wrappers into a single wrapper.
// Wrappers are executed in the order they are passed.
// Example: Chain(W1, W2) results in W1(W2(Job)).
func Chain(wrappers ...JobWrapper) JobWrapper {
	return func(next JobCmd) JobCmd {
		for i := len(wrappers) - 1; i >= 0; i-- {
			next = wrappers[i](next)
		}
		return next
	}
}

// Recover creates a wrapper that catches panics and returns them as errors.
func Recover() JobWrapper {
	return func(next JobCmd) JobCmd {
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

// DelayIfStillRunning ensures that job executions are sequential.
// If a job is triggered while a previous instance is running, it waits.
func DelayIfStillRunning() JobWrapper {
	var mu sync.Mutex
	return func(next JobCmd) JobCmd {
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
	return func(next JobCmd) JobCmd {
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
