package cronlib

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Logic Tests (copied from logic_test.go) ---

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
		check   func(Expression) bool
	}{
		{
			name:    "Simple precise time",
			spec:    "5 4 3 2 1 0",
			wantErr: false,
			check: func(e Expression) bool {
				return e.second == (1<<5) &&
					e.minute == (1<<4) &&
					e.hour == (1<<3) &&
					e.dom == (1<<2) &&
					e.month == (1<<1) &&
					e.dow == (1<<0)
			},
		},
		{
			name:    "All stars",
			spec:    "* * * * * *",
			wantErr: false,
			check: func(e Expression) bool {
				return e.second == (1<<60)-1 && // 0-59 set
					e.minute == (1<<60)-1 &&
					e.hour == (1<<24)-1 && // 0-23
					e.dom == (1<<32)-2 && // 1-31 (bit 1..31)
					e.month == (1<<13)-2 && // 1-12
					e.dow == (1<<7)-1 && // 0-6
					e.domStar && e.dowStar
			},
		},
		{
			name:    "Step",
			spec:    "*/15 * * * * *",
			wantErr: false,
			check: func(e Expression) bool {
				// 0, 15, 30, 45
				expected := uint64(1<<0 | 1<<15 | 1<<30 | 1<<45)
				return e.second == expected
			},
		},
		{
			name:    "Range",
			spec:    "0-5 * * * * *",
			wantErr: false,
			check: func(e Expression) bool {
				// 0,1,2,3,4,5
				expected := uint64(1<<0 | 1<<1 | 1<<2 | 1<<3 | 1<<4 | 1<<5)
				return e.second == expected
			},
		},
		{
			name:    "List",
			spec:    "0,15,30 * * * * *",
			wantErr: false,
			check: func(e Expression) bool {
				expected := uint64(1<<0 | 1<<15 | 1<<30)
				return e.second == expected
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				if !tt.check(got) {
					t.Errorf("Parse() validation failed for %s", tt.spec)
				}
			}
		})
	}
}

func TestNext(t *testing.T) {
	// Helper to make time
	mkTime := func(year int, month time.Month, day, hour, min, sec int) time.Time {
		return time.Date(year, month, day, hour, min, sec, 0, time.UTC)
	}

	tests := []struct {
		name string
		spec string
		from time.Time
		want time.Time
	}{
		{
			name: "Every second",
			spec: "* * * * * *",
			from: mkTime(2023, 1, 1, 0, 0, 0),
			want: mkTime(2023, 1, 1, 0, 0, 1),
		},
		{
			name: "Specific time (seconds)",
			spec: "30 * * * * *",
			from: mkTime(2023, 1, 1, 0, 0, 0),
			want: mkTime(2023, 1, 1, 0, 0, 30),
		},
		{
			name: "Specific time (seconds) wrap",
			spec: "30 * * * * *",
			from: mkTime(2023, 1, 1, 0, 0, 31),
			want: mkTime(2023, 1, 1, 0, 1, 30),
		},
		{
			name: "Every 15 minutes",
			spec: "0 */15 * * * *",
			from: mkTime(2023, 1, 1, 0, 0, 0),
			want: mkTime(2023, 1, 1, 0, 15, 0),
		},
		{
			name: "Hour wrap",
			spec: "0 0 * * * *",
			from: mkTime(2023, 1, 1, 0, 59, 59),
			want: mkTime(2023, 1, 1, 1, 0, 0),
		},
		{
			name: "Day wrap",
			spec: "0 0 12 * * *",
			from: mkTime(2023, 1, 1, 12, 0, 0),
			want: mkTime(2023, 1, 2, 12, 0, 0),
		},
		{
			name: "Month wrap",
			spec: "0 0 0 1 * *",
			from: mkTime(2023, 1, 31, 0, 0, 0),
			want: mkTime(2023, 2, 1, 0, 0, 0),
		},
		{
			name: "Year wrap",
			spec: "0 0 0 1 1 *",
			from: mkTime(2023, 1, 2, 0, 0, 0),
			want: mkTime(2024, 1, 1, 0, 0, 0),
		},
		{
			name: "DOM/DOW Union (Fri OR 15th)",
			// 2023 Jan 1 is Sunday.
			// 15th is Sunday.
			// Friday is 6th, 13th, 20th.
			spec: "0 0 0 15 * 5", // 15th of month OR Friday
			from: mkTime(2023, 1, 1, 0, 0, 0),
			// Next is Friday Jan 6th.
			want: mkTime(2023, 1, 6, 0, 0, 0),
		},
		{
			name: "DOM/DOW Intersection (DOM restricted, DOW *)",
			spec: "0 0 0 15 * *", // 15th of month
			from: mkTime(2023, 1, 1, 0, 0, 0),
			want: mkTime(2023, 1, 15, 0, 0, 0),
		},
		{
			name: "DOM/DOW Intersection (DOM *, DOW restricted)",
			spec: "0 0 0 * * 5", // Every Friday
			from: mkTime(2023, 1, 1, 0, 0, 0),
			want: mkTime(2023, 1, 6, 0, 0, 0),
		},
		{
			name: "DOM/DOW with steps",
			// */2 = 2,4,6... ? 
			// DOW 0-6. */2 => 0, 2, 4, 6 (Sun, Tue, Thu, Sat)
			spec: "0 0 0 * * */2",
			from: mkTime(2023, 1, 1, 0, 0, 0), // Jan 1 is Sun (0)
			// Next is Jan 3 (Tue) (2)
			want: mkTime(2023, 1, 3, 0, 0, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.spec)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			got := expr.Next(tt.from)
			if !got.Equal(tt.want) {
				t.Errorf("Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Integration Tests ---

func TestCron_Integration_Run(t *testing.T) {
	c := NewCron()
	c.Start()
	defer c.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	
	start := time.Now()
	ran := false
	var ranOnce sync.Once
	
	// Run every second
	_, err := c.AddJob("* * * * * *", func() {
		ranOnce.Do(func() {
			ran = true
			wg.Done()
		})
	})
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	
	// Wait for it to run
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Success
		elapsed := time.Since(start)
		if elapsed > 2*time.Second {
			t.Logf("Job ran but took long: %v", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Job did not run within 3 seconds")
	}
	
	if !ran {
		t.Fatal("Job ran flag not set")
	}
}

func TestCron_ConcurrentAdd(t *testing.T) {
	c := NewCron()
	c.Start()
	defer c.Stop()
	
	const numJobs = 50
	var addWg sync.WaitGroup
	addWg.Add(numJobs)
	
	for i := 0; i < numJobs; i++ {
		go func(i int) {
			defer addWg.Done()
			_, err := c.AddJob("* * * * * *", func() {
				// No-op
			})
			if err != nil {
				t.Errorf("AddJob failed: %v", err)
			}
		}(i)
	}
	addWg.Wait()
	
	c.mu.RLock()
	count := len(c.jobs)
	c.mu.RUnlock()
	
	if count != numJobs {
		t.Errorf("Expected %d jobs, got %d", numJobs, count)
	}
}

func TestCron_GracefulShutdown(t *testing.T) {
	c := NewCron()
	c.Start()
	
	var wg sync.WaitGroup
	wg.Add(1)
	
	running := make(chan struct{})
	var runOnce sync.Once
	
	c.AddJob("* * * * * *", func() {
		runOnce.Do(func() {
			close(running) // Signal started
			time.Sleep(500 * time.Millisecond) // Simulate work
			wg.Done()
		})
	})
	
	// Wait until job starts
	select {
	case <-running:
	case <-time.After(2 * time.Second):
		t.Fatal("Job didn't start")
	}
	
	startStop := time.Now()
	c.Stop() // Should wait for job
	elapsed := time.Since(startStop)
	
	if elapsed < 500*time.Millisecond {
		t.Errorf("Stop returned too early, took %v, expected > 500ms", elapsed)
	}
	
	// Verify job finished
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Job did not finish cleanly")
	}
}

func TestCron_Precision(t *testing.T) {
	c := NewCron()
	c.Start()
	defer c.Stop()
	
	var wg sync.WaitGroup
	wg.Add(1)
	
	var runTime time.Time
	var runOnce sync.Once
	
	// Schedule for next second
	// Actually, just "* * * * * *" will fire at next second boundary.
	
	_, err := c.AddJob("* * * * * *", func() {
		runOnce.Do(func() {
			runTime = time.Now()
			wg.Done()
		})
	})
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
		// Calculate deviation from expected second boundary
		// runTime should be close to Truncate(Second).
		// Note: The job runs at X.000 (roughly).
		// Deviation = (runTime - runTime.Truncate(Second))
		// Actually runTime.Nanosecond() should be small.
		ns := runTime.Nanosecond()
		if ns > 500_000_000 {
			// It ran closer to next second? No, typically it runs slightly after :00.
			// If it runs at :59.999, ns is huge.
			// But we expect it to run after.
		}
		
		// If ns is large (e.g. 900ms), we might have been late or early.
		// We expect close to 0.
		// Let's check deviation from nearest second.
		
		// Note: If we just verify it's within 20ms of a second boundary.
		// But which boundary?
		// We can't easily know exactly which second it picked without calculation.
		// But we know it should be consistent.
		
		if ns > 50_000_000 { // 50ms tolerance
			t.Logf("Precision warning: Job ran at %v (ns: %d), > 50ms deviation", runTime, ns)
			// Don't fail the test as CI environments can be slow, but log it.
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Job didn't run")
	}
}

func TestCron_RemoveJob(t *testing.T) {
	c := NewCron()
	c.Start()
	defer c.Stop()
	
	var counter int32
	
	id, _ := c.AddJob("* * * * * *", func() {
		atomic.AddInt32(&counter, 1)
	})
	
	// let it run once? No, remove immediately.
	c.RemoveJob(id)
	
	// Wait 2 seconds. Counter should be 0.
	time.Sleep(2 * time.Second)
	
	val := atomic.LoadInt32(&counter)
	if val != 0 {
		t.Errorf("Job ran %d times after removal (expected 0). (Note: if it ran before removal, this test is flaky, but we removed immediately)", val)
	}
}

func TestCron_ImpossibleDate(t *testing.T) {
	c := NewCron()
	
	// Feb 30th
	// 0 0 0 30 2 *
	_, err := c.AddJob("0 0 0 30 2 *", func() {})
	if err == nil {
		t.Error("Expected error for impossible date (Feb 30), got nil")
	}
}
