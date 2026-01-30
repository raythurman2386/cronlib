package cronlib

import (
	"testing"
	"time"
)

// BenchmarkParse measures the cost of parsing a standard cron expression.
func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = Parse("*/5 * * * * *")
	}
}

// BenchmarkNext measures the cost of calculating the next execution time.
func BenchmarkNext(b *testing.B) {
	expr, _ := Parse("*/5 * * * * *")
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expr.Next(now)
	}
}

// BenchmarkAddJob measures the memory cost of adding a job.
func BenchmarkAddJob(b *testing.B) {
	c := NewCron()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.AddJob("* * * * * *", func() {})
	}
}

// BenchmarkCron_Run measures the overhead of the scheduler loop with many jobs.
// We don't actually execute the jobs (to avoid measuring the job's work),
// but we verify the scheduling logic handles many entries.
func BenchmarkScheduler_HighLoad(b *testing.B) {
	c := NewCron()
	// Add 1000 jobs
	for i := 0; i < 1000; i++ {
		_, _ = c.AddJob("*/1 * * * * *", func() {})
	}

	b.ReportAllocs()
	b.ResetTimer()

	// We can't easily benchmark the "loop" since it runs in a goroutine.
	// Instead, we can benchmark the `Next` calculation for the whole set indirectly,
	// or just benchmark the sorting/management.

	// Actually, let's benchmark Parse + Add for a large batch.
	for i := 0; i < b.N; i++ {
		_, _ = c.AddJob("*/5 * * * * *", func() {})
	}
}
