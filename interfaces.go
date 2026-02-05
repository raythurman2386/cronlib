package cronlib

import (
	"context"
	"time"
)

// JobCmd is the function signature for a job's command.
type JobCmd = func(context.Context) error

// JobStore defines persistence for job state and history.
type JobStore interface {
	GetLastRun(jobID string) (time.Time, error)
	SetLastRun(jobID string, t time.Time) error
	LogExecution(jobID string, start, end time.Time, success bool, out string) error
}

// DistLock defines distributed locking behavior.
type DistLock interface {
	// Lock attempts to acquire a lock for the key. Returns true if acquired.
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// Unlock releases the lock.
	Unlock(ctx context.Context, key string) error
}

// OverlapPolicy defines how to handle job overlaps.
type OverlapPolicy int

const (
	// OverlapAllow allows multiple instances of the same job to run concurrently.
	OverlapAllow OverlapPolicy = iota
	// OverlapForbid skips execution if the previous instance is still running.
	OverlapForbid
	// OverlapReplace cancels the running instance and starts a new one.
	OverlapReplace
)

// JobWrapper decorates a job execution function.
// It allows injecting custom logic before and after the job runs (middleware).
type JobWrapper = func(JobCmd) JobCmd
