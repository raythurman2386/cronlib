package cronlib

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Cron is the main scheduler engine.
// It manages a list of jobs and executes them according to their schedule.
// Methods on Cron are thread-safe.
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

	loc := opts.Location
	if loc == nil {
		loc = time.Local
	}

	// Check store for last run
	var lastRun time.Time
	if c.store != nil {
		lr, err := c.store.GetLastRun(id)
		if err == nil && !lr.IsZero() {
			lastRun = lr.In(loc)
		}
	}

	var next time.Time
	if !lastRun.IsZero() {
		next = expr.Next(lastRun)
	} else {
		next = expr.Next(time.Now().In(loc))
	}

	if next.IsZero() {
		return "", fmt.Errorf("impossible schedule")
	}

	// Adapt user cmd and apply wrappers
	baseCmd := func(ctx context.Context) error {
		cmd(ctx)
		return nil
	}

	wrappers := []JobWrapper{
		Recover(),
		c.lockWrapper(id),
		c.logWrapper(id),
	}

	wrappers = append(wrappers, opts.Wrappers...)

	finalCmd := Chain(wrappers...)(baseCmd)

	job := &Job{
		ID:   id,
		Spec: spec,
		Expr: expr,
		Cmd:  finalCmd,
		next: next,
		opts: opts,
		loc:  loc,
	}

	c.jobs[id] = job
	c.jobList = append(c.jobList, job)

	// Keep sorted by next time
	sort.Slice(c.jobList, func(i, j int) bool {
		return c.jobList[i].next.Before(c.jobList[j].next)
	})

	if c.running {
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

	for i, j := range c.jobList {
		if j == job {
			copy(c.jobList[i:], c.jobList[i+1:])
			c.jobList[len(c.jobList)-1] = nil
			c.jobList = c.jobList[:len(c.jobList)-1]
			break
		}
	}
}

// Start starts the scheduler loop.
func (c *Cron) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	c.wg.Add(1)
	go c.run()
}

// Stop stops the scheduler.
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
		if len(c.jobList) == 0 {
			delay = 100000 * time.Hour
		} else {
			delay = time.Until(c.jobList[0].next)
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
			timer.Stop()

		case <-timer.C:
			now := time.Now()

			c.mu.Lock()
			for len(c.jobList) > 0 {
				job := c.jobList[0]
				if job.next.After(now.Add(10 * time.Millisecond)) {
					break
				}

				job.mu.Lock()
				if job.running {
					switch job.opts.Overlap {
					case OverlapForbid:
						job.mu.Unlock()
						goto AdvanceJob
					case OverlapReplace:
						if job.cancel != nil {
							job.cancel()
						}
					case OverlapAllow:
					}
				}
				job.mu.Unlock()

				c.wg.Add(1)
				go func(j *Job) {
					defer c.wg.Done()

					j.mu.Lock()
					j.running = true
					ctx, cancel := context.WithCancel(context.Background())
					j.cancel = cancel
					j.mu.Unlock()

					_ = j.Cmd(ctx)

					j.mu.Lock()
					j.running = false
					if j.cancel != nil {
						j.cancel()
						j.cancel = nil
					}
					j.mu.Unlock()
				}(job)

			AdvanceJob:
				next := job.Expr.Next(now.In(job.loc))
				if next.IsZero() {
					delete(c.jobs, job.ID)
					c.jobList[0] = nil
					c.jobList = c.jobList[1:]
					continue
				}
				job.next = next

				sort.Slice(c.jobList, func(i, j int) bool {
					return c.jobList[i].next.Before(c.jobList[j].next)
				})
			}
			c.mu.Unlock()
		}
	}
}

// Internal wrappers that use Cron state

func (c *Cron) lockWrapper(id string) JobWrapper {
	return func(next JobCmd) JobCmd {
		return func(ctx context.Context) error {
			c.mu.RLock()
			lock := c.lock
			c.mu.RUnlock()

			if lock == nil {
				return next(ctx)
			}

			key := "cronlib:lock:" + id
			ok, err := lock.Lock(ctx, key, 1*time.Hour)
			if err != nil {
				return err
			}
			if !ok {
				return nil // Locked by another instance
			}
			defer func() { _ = lock.Unlock(ctx, key) }()

			return next(ctx)
		}
	}
}

func (c *Cron) logWrapper(id string) JobWrapper {
	return func(next JobCmd) JobCmd {
		return func(ctx context.Context) error {
			c.mu.RLock()
			store := c.store
			c.mu.RUnlock()

			start := time.Now()
			err := next(ctx)
			end := time.Now()

			if store != nil {
				_ = store.SetLastRun(id, start)
				out := ""
				if err != nil {
					out = err.Error()
				}
				_ = store.LogExecution(id, start, end, err == nil, out)
			}

			return err
		}
	}
}
