package cronlib

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

// JobOptions configuration for a job.
type JobOptions struct {
	Overlap OverlapPolicy
}

// Expression represents the parsed cron expression using bitmasks.
type Expression struct {
	second uint64
	minute uint64
	hour   uint64
	dom    uint64
	month  uint64
	dow    uint64
	// Flags for standard cron behavior
	domStar bool
	dowStar bool
}

// Job represents a scheduled task.
// Job represents a scheduled task.
type Job struct {
	ID   string
	Spec string
	Expr Expression
	Cmd  func(context.Context)
	next time.Time

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	opts    JobOptions
}

// Cron is the main scheduler engine.
type Cron struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	jobList []*Job // Kept sorted by next run time
	running bool
	stopCh  chan struct{}
	addCh   chan *Job // Signal to re-evaluate schedule
	wg      sync.WaitGroup
	idCtr   int

	store JobStore
	lock  DistLock
}

// NewCron creates a new Cron scheduler.
func NewCron() *Cron {
	return &Cron{
		jobs:    make(map[string]*Job),
		jobList: make([]*Job, 0),
		stopCh:  make(chan struct{}),
		addCh:   make(chan *Job, 1),
	}
}

// SetJobStore configures the persistence layer.
func (c *Cron) SetJobStore(s JobStore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = s
}

// SetDistLock configures the distributed lock.
func (c *Cron) SetDistLock(l DistLock) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lock = l
}

// Parse parses a 6-field cron string into an Expression.
// Format: second minute hour day-of-month month day-of-week
func Parse(spec string) (Expression, error) {
	fields := strings.Fields(spec)
	if len(fields) != 6 {
		return Expression{}, fmt.Errorf("expected 6 fields, found %d: %s", len(fields), spec)
	}

	var err error
	var expr Expression

	expr.second, _, err = parseField(fields[0], 0, 59)
	if err != nil {
		return expr, fmt.Errorf("parsing second: %w", err)
	}

	expr.minute, _, err = parseField(fields[1], 0, 59)
	if err != nil {
		return expr, fmt.Errorf("parsing minute: %w", err)
	}

	expr.hour, _, err = parseField(fields[2], 0, 23)
	if err != nil {
		return expr, fmt.Errorf("parsing hour: %w", err)
	}

	expr.dom, expr.domStar, err = parseField(fields[3], 1, 31)
	if err != nil {
		return expr, fmt.Errorf("parsing dom: %w", err)
	}

	expr.month, _, err = parseField(fields[4], 1, 12)
	if err != nil {
		return expr, fmt.Errorf("parsing month: %w", err)
	}

	expr.dow, expr.dowStar, err = parseField(fields[5], 0, 6)
	if err != nil {
		return expr, fmt.Errorf("parsing dow: %w", err)
	}

	return expr, nil
}

func parseField(field string, min, max int) (uint64, bool, error) {
	var bits uint64

	// Handle list
	parts := strings.Split(field, ",")
	for _, part := range parts {
		if part == "*" {
			for i := min; i <= max; i++ {
				bits |= (1 << i)
			}
			continue
		}

		// Handle step
		step := 1
		rangePart := part
		if i := strings.Index(part, "/"); i >= 0 {
			stepStr := part[i+1:]
			var err error
			step, err = strconv.Atoi(stepStr)
			if err != nil || step <= 0 {
				return 0, false, fmt.Errorf("invalid step: %s", part)
			}
			rangePart = part[:i]
		}

		// Handle range
		var start, end int
		if rangePart == "*" {
			start = min
			end = max
			// */n is not "star" in the sense of "unrestricted"
			// But for dom/dow behavior, usually only pure "*" counts as unrestricted.
			// However, parseField logic above sets isStar=true for "*".
			// If we have "*/2", rangePart is "*", so we treat it as range min-max.
			// Is "*/2" considered "restricted"? Yes.
			// So if we have a step, it's restricted.
			// Correct logic: isStar should be true ONLY if field is exactly "*"
		} else if i := strings.Index(rangePart, "-"); i >= 0 {
			startStr := rangePart[:i]
			endStr := rangePart[i+1:]
			var err error
			start, err = strconv.Atoi(startStr)
			if err != nil {
				return 0, false, fmt.Errorf("invalid range start: %s", rangePart)
			}
			end, err = strconv.Atoi(endStr)
			if err != nil {
				return 0, false, fmt.Errorf("invalid range end: %s", rangePart)
			}
		} else {
			// Single number
			val, err := strconv.Atoi(rangePart)
			if err != nil {
				return 0, false, fmt.Errorf("invalid number: %s", rangePart)
			}
			start = val
			end = val
		}

		if start < min || end > max {
			return 0, false, fmt.Errorf("value out of range (%d-%d): %s", min, max, part)
		}

		for i := start; i <= end; i += step {
			if i > max {
				break
			}
			bits |= (1 << i)
		}
	}

	if bits == 0 {
		return 0, false, fmt.Errorf("no valid values in field: %s", field)
	}

	// Refined star check: if the input string is exactly "*", it's a star.
	// But "*,*" or "*/1" etc.
	// Simple rule: field == "*"
	return bits, field == "*", nil
}

// Next returns the next execution time after `from`.
// Returns zero time if no match found within 5 years.
func (e Expression) Next(from time.Time) time.Time {
	// Start checking from the next second
	t := from.Add(1 * time.Second)
	// Strip nanoseconds
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())

	for i := 0; i < 5*366*24*60*60; i++ { // Safety limit for iterations
		if t.Year()-from.Year() > 5 {
			return time.Time{}
		}

		// 1. Month
		if (1<<int(t.Month()))&e.month == 0 {
			// Find next valid month
			nextMonth, ok := findNextBit(e.month, int(t.Month())+1, 1, 12)
			if !ok {
				// Next year
				nextMonth, _ = findNextBit(e.month, 1, 1, 12)
				t = time.Date(t.Year()+1, time.Month(nextMonth), 1, 0, 0, 0, 0, t.Location())
			} else {
				t = time.Date(t.Year(), time.Month(nextMonth), 1, 0, 0, 0, 0, t.Location())
			}
			continue
		}

		// 2. Day (DOM/DOW)
		// Logic:
		// If neither DOM nor DOW is *, match if EITHER matches.
		// If one is *, match if the OTHER matches.
		// (If both are *, it matches every day).
		domMatch := (1<<uint(t.Day()))&e.dom != 0
		dowMatch := (1<<uint(t.Weekday()))&e.dow != 0

		dayMatched := false
		if !e.domStar && !e.dowStar {
			dayMatched = domMatch || dowMatch
		} else if e.domStar {
			dayMatched = dowMatch
		} else if e.dowStar {
			dayMatched = domMatch
		} else {
			dayMatched = true
		}

		if !dayMatched {
			// Increment day
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}

		// 3. Hour
		if (1<<uint(t.Hour()))&e.hour == 0 {
			nextHour, ok := findNextBit(e.hour, t.Hour()+1, 0, 23)
			if ok {
				t = time.Date(t.Year(), t.Month(), t.Day(), nextHour, 0, 0, 0, t.Location())
			} else {
				// Next day
				t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			}
			continue
		}

		// 4. Minute
		if (1<<uint(t.Minute()))&e.minute == 0 {
			nextMinute, ok := findNextBit(e.minute, t.Minute()+1, 0, 59)
			if ok {
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), nextMinute, 0, 0, t.Location())
			} else {
				// Next hour
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			}
			continue
		}

		// 5. Second
		if (1<<uint(t.Second()))&e.second == 0 {
			nextSecond, ok := findNextBit(e.second, t.Second()+1, 0, 59)
			if ok {
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), nextSecond, 0, t.Location())
			} else {
				// Next minute
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, t.Location())
			}
			continue
		}

		// If we are here, everything matched
		return t
	}

	return time.Time{}
}

// findNextBit returns the index of the next set bit in mask >= start.
// min/max define the range of the field (to bounds check).
func findNextBit(mask uint64, start, min, max int) (int, bool) {
	for i := start; i <= max; i++ {
		if (mask & (1 << i)) != 0 {
			return i, true
		}
	}
	return 0, false
}

// AddJob adds a new job to the scheduler.
// Returns the job ID and error if spec is invalid.
// AddJob adds a new job to the scheduler.
// Returns the job ID and error if spec is invalid.
func (c *Cron) AddJob(spec string, cmd func()) (string, error) {
	return c.AddJobWithOptions(spec, func(ctx context.Context) { cmd() }, JobOptions{Overlap: OverlapAllow})
}

// AddJobWithOptions adds a new job with specific options.
func (c *Cron) AddJobWithOptions(spec string, cmd func(context.Context), opts JobOptions) (string, error) {
	expr, err := Parse(spec)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.idCtr++
	id := strconv.Itoa(c.idCtr)

	// Check store for last run
	var lastRun time.Time
	if c.store != nil {
		lr, err := c.store.GetLastRun(id)
		if err == nil && !lr.IsZero() {
			lastRun = lr
		}
	}

	var next time.Time
	if !lastRun.IsZero() {
		next = expr.Next(lastRun)
	} else {
		next = expr.Next(time.Now())
	}

	if next.IsZero() {
		return "", fmt.Errorf("impossible schedule")
	}

	job := &Job{
		ID:   id,
		Spec: spec,
		Expr: expr,
		Cmd:  cmd,
		next: next,
		opts: opts,
	}

	c.jobs[id] = job
	c.jobList = append(c.jobList, job)

	// Keep sorted by next time
	sort.Slice(c.jobList, func(i, j int) bool {
		return c.jobList[i].next.Before(c.jobList[j].next)
	})

	if c.running {
		// Signal scheduler to pick up the new job if it's the soonest
		// Use non-blocking send or buffered
		select {
		case c.addCh <- job:
		default:
		}
	}

	return id, nil
}

// RemoveJob removes a job by ID.
func (c *Cron) RemoveJob(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	job, ok := c.jobs[id]
	if !ok {
		return
	}

	delete(c.jobs, id)

	// Remove from list
	for i, j := range c.jobList {
		if j == job {
			copy(c.jobList[i:], c.jobList[i+1:])
			c.jobList[len(c.jobList)-1] = nil
			c.jobList = c.jobList[:len(c.jobList)-1]
			break
		}
	}
	// No need to signal, scheduler will wake up and re-check list
}

// Start starts the scheduler loop.
func (c *Cron) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.stopCh = make(chan struct{}) // Re-create stop channel
	c.mu.Unlock()

	c.wg.Add(1)
	go c.run()
}

// Stop stops the scheduler and waits for running jobs to finish.
func (c *Cron) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	close(c.stopCh)
	c.mu.Unlock()

	c.wg.Wait()
}

func (c *Cron) run() {
	defer c.wg.Done()

	for {
		c.mu.RLock()
		var delay time.Duration
		var nextTime time.Time
		if len(c.jobList) == 0 {
			delay = 100000 * time.Hour // "Infinite" wait
		} else {
			nextTime = c.jobList[0].next
			delay = nextTime.Sub(time.Now())
		}
		c.mu.RUnlock()

		if delay < 0 {
			delay = 0
		}

		timer := time.NewTimer(delay)

		select {
		case <-c.stopCh:
			timer.Stop()
			return

		case <-c.addCh:
			// New job added, wake up and recalculate
			timer.Stop()
			// Loop will re-run

		case <-timer.C:
			// Timer fired.
			now := time.Now()

			c.mu.Lock()
			// Process all jobs that are due
			for len(c.jobList) > 0 {
				job := c.jobList[0]
				if job.next.After(now.Add(10 * time.Millisecond)) {
					break
				}

				// Check overlap policy
				job.mu.Lock()
				if job.running {
					switch job.opts.Overlap {
					case OverlapForbid:
						// Skip execution
						job.mu.Unlock()
						// Still need to update next run time!
						goto AdvanceJob
					case OverlapReplace:
						// Cancel previous
						if job.cancel != nil {
							job.cancel()
						}
						// Don't wait, proceed to run new instance
					case OverlapAllow:
						// Default, do nothing
					}
				}
				job.mu.Unlock()

				// Execute job
				c.wg.Add(1)
				go func(j *Job) {
					defer c.wg.Done()

					// Distributed Lock
					if c.lock != nil {
						// 1 minute TTL for now
						locked, err := c.lock.Lock(context.Background(), "cron:"+j.ID, time.Minute)
						if err != nil || !locked {
							// Failed to acquire lock, skip execution
							return
						}
						defer c.lock.Unlock(context.Background(), "cron:"+j.ID)
					}

					// Mark running
					j.mu.Lock()
					j.running = true
					ctx, cancel := context.WithCancel(context.Background())
					j.cancel = cancel
					j.mu.Unlock()

					start := time.Now()
					var err error
					defer func() {
						end := time.Now()
						j.mu.Lock()
						j.running = false
						if j.cancel != nil {
							j.cancel() // Ensure cleanup
							j.cancel = nil
						}
						j.mu.Unlock()

						// Log execution
						if c.store != nil {
							success := err == nil
							msg := ""
							if err != nil {
								msg = err.Error()
							}
							c.store.LogExecution(j.ID, start, end, success, msg)
							c.store.SetLastRun(j.ID, start)
						}
					}()

					// Catch panics? Ideally yes, but keeping it simple for now
					j.Cmd(ctx)
				}(job)

			AdvanceJob:
				// Calculate next run
				next := job.Expr.Next(now)
				if next.IsZero() {
					delete(c.jobs, job.ID)
					c.jobList[0] = nil
					c.jobList = c.jobList[1:]
					continue
				}
				job.next = next

				// Re-sort list
				sort.Slice(c.jobList, func(i, j int) bool {
					return c.jobList[i].next.Before(c.jobList[j].next)
				})
			}
			c.mu.Unlock()
		}
	}
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
