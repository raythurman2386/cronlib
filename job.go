package cronlib

import (
	"context"
	"sync"
	"time"
)

// JobOptions configuration for a job.
type JobOptions struct {
	Overlap  OverlapPolicy
	Location *time.Location
	Wrappers []JobWrapper
}

// Job represents a scheduled task.
type Job struct {
	ID   string
	Spec string
	Expr Expression
	Cmd  func(context.Context) error
	next time.Time

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	opts    JobOptions
	loc     *time.Location
}

// JobStatus represents the state of a job.
type JobStatus struct {
	ID         string    `json:"id"`
	Expression string    `json:"expression"`
	NextRun    time.Time `json:"next_run"`
	LastRun    time.Time `json:"last_run"`
	Running    bool      `json:"running"`
}

// GetJobs returns the status of all scheduled jobs.
func (c *Cron) GetJobs() []JobStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var stats []JobStatus
	for _, job := range c.jobList {
		lastRun := time.Time{}
		if c.store != nil {
			// Best effort to get last run
			lr, _ := c.store.GetLastRun(job.ID)
			lastRun = lr
		}

		stats = append(stats, JobStatus{
			ID:         job.ID,
			Expression: job.Spec,
			NextRun:    job.next,
			LastRun:    lastRun,
			Running:    job.running,
		})
	}
	return stats
}
