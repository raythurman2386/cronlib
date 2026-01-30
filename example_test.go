package cronlib_test

import (
	"context"
	"fmt"
	"time"

	"github.com/raythurman2386/cronlib"
)

func ExampleCron_AddJob() {
	c := cronlib.NewCron()

	// Run every second
	// Note: We use a channel to ensure the example output is deterministic
	// for the purpose of this testable example.
	done := make(chan struct{})

	_, err := c.AddJob("* * * * * *", func() {
		fmt.Println("Job executed")
		close(done)
	})
	if err != nil {
		fmt.Println("Error scheduling job:", err)
		return
	}

	c.Start()
	defer c.Stop()

	<-done
	// Output: Job executed
}

func ExampleCron_AddJobWithOptions() {
	c := cronlib.NewCron()

	// Define a job with specific options
	opts := cronlib.JobOptions{
		Overlap: cronlib.OverlapForbid, // Skip if previous run is still active
	}

	_, err := c.AddJobWithOptions("*/5 * * * * *", func(ctx context.Context) {
		fmt.Println("Job with context running")
	}, opts)
	if err != nil {
		fmt.Println("Error:", err)
	}

	c.Start()
	// Allow time for execution in a real app
	time.Sleep(1 * time.Second)
	c.Stop()
}

func ExampleChain() {
	// Define a custom wrapper that logs execution
	loggingWrapper := func(next func(context.Context) error) func(context.Context) error {
		return func(ctx context.Context) error {
			fmt.Println("Starting job...")
			err := next(ctx)
			fmt.Println("Job finished")
			return err
		}
	}

	c := cronlib.NewCron()
	opts := cronlib.JobOptions{
		Wrappers: []cronlib.JobWrapper{loggingWrapper},
	}

	c.AddJobWithOptions("@every 1s", func(ctx context.Context) {
		fmt.Println("Doing work")
	}, opts)

	c.Start()
	// Wait for one run
	time.Sleep(1100 * time.Millisecond)
	c.Stop()
}
