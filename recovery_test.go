package cronlib

import (
	"sync"
	"testing"
	"time"
)

func TestPanicRecovery(t *testing.T) {
	c := NewCron()
	var wg sync.WaitGroup
	wg.Add(1)

	// Add a job that panics
	// Run every second
	_, err := c.AddJob("* * * * * *", func() {
		defer wg.Done()
		panic("YIKES")
	})
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	c.Start()
	defer c.Stop()

	// Wait for the job to run (and panic)
	// If the panic is not recovered, the test binary should crash here.
	// We use a channel to timeout if it takes too long (though crash is immediate).
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Job ran and panicked. If we are here, the test didn't crash.
		// Wait a bit more to ensure scheduler is still alive
		time.Sleep(100 * time.Millisecond)
		if !c.running {
			t.Error("Scheduler stopped running after panic")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for job to run")
	}
}
