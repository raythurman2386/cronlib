package cronlib

import (
	"context"
	"testing"
	"time"
)

func TestTimezoneSupport(t *testing.T) {
	c := NewCron()

	// Define a fixed zone: UTC+2
	loc := time.FixedZone("UTC+2", 2*60*60)

	// Current time in UTC
	nowUTC := time.Now().UTC()
	// Corresponding time in UTC+2
	nowInLoc := nowUTC.In(loc)

	// We want to schedule a job for 1 hour from "now" in the custom timezone.
	// For example, if it's 12:00 UTC (14:00 UTC+2), we schedule for 15:00 UTC+2.
	// That is 13:00 UTC.

	targetHour := nowInLoc.Hour() + 1
	if targetHour >= 24 {
		targetHour -= 24
	}

	spec := "0 0 " + getStr(targetHour) + " * * *"

	// Add job with location
	id, err := c.AddJobWithOptions(spec, func(ctx context.Context) {}, JobOptions{
		Location: loc,
	})
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	// Check the calculated next run time
	jobs := c.GetJobs()
	var job JobStatus
	found := false
	for _, j := range jobs {
		if j.ID == id {
			job = j
			found = true
			break
		}
	}

	if !found {
		t.Fatal("Job not found")
	}

	// Verify the next run time's location
	if job.NextRun.Location().String() != loc.String() {
		t.Errorf("Expected location %v, got %v", loc, job.NextRun.Location())
	}

	// The NextRun time should be correct.
	// Check that the hour matches our target hour in that timezone
	if job.NextRun.In(loc).Hour() != targetHour {
		t.Errorf("Expected next run hour %d (in %v), got %d. Next run: %v", targetHour, loc, job.NextRun.In(loc).Hour(), job.NextRun)
	}
}

func getStr(i int) string {
	if i < 10 {
		return "0" + string(rune('0'+i))
	}
	return string(rune('0'+(i/10))) + string(rune('0'+(i%10)))
}
