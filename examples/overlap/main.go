package main

import (
	"context"
	"fmt"
	"time"

	"github.com/raythurman2386/cronlib"
)

func main() {
	c := cronlib.NewCron()

	// Use Case: A slow database cleanup task scheduled every 2 seconds.
	// But the cleanup itself takes 4 seconds.

	// 1. Overlap Forbid: Skip if previous is still running (Good for preventing load)
	c.AddJobWithOptions("*/2 * * * * *", func(ctx context.Context) {
		fmt.Println("[FORBID]  Starting cleanup (takes 4s)...")
		time.Sleep(4 * time.Second)
		fmt.Println("[FORBID]  Cleanup finished.")
	}, cronlib.JobOptions{Overlap: cronlib.OverlapForbid})

	// 2. Overlap Replace: Cancel previous and start fresh (Good for real-time dashboards)
	c.AddJobWithOptions("*/2 * * * * *", func(ctx context.Context) {
		fmt.Println("[REPLACE] Starting fresh poll...")
		select {
		case <-time.After(4 * time.Second):
			fmt.Println("[REPLACE] Poll finished normally.")
		case <-ctx.Done():
			fmt.Println("[REPLACE] !! Current poll cancelled by new one !!")
		}
	}, cronlib.JobOptions{Overlap: cronlib.OverlapReplace})

	fmt.Println("Overlap Control simulation starting...")
	fmt.Println("Watch how [FORBID] skips every other run, while [REPLACE] never finishes a poll.")

	c.Start()
	time.Sleep(15 * time.Second)
	c.Stop()
}
