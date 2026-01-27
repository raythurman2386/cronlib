package main

import (
	"fmt"
	"log"
	"time"

	"github.com/raythurman2386/cronlib"
	"github.com/raythurman2386/cronlib/store/sqlite"
)

func main() {
	// 1. Initialize SQLite Store
	// This store keeps track of the 'LastRun' time for every job.
	store, err := sqlite.New("recovery.db")
	if err != nil {
		log.Fatalf("Failed to open store: %v", err)
	}
	defer store.Close()

	c := cronlib.NewCron()
	c.SetJobStore(store)

	// 2. Add a persistent job
	// When AddJob is called, CronLib checks recovery.db to see when it last ran.
	// This prevents duplicate runs if the app is restarted frequently.
	id, _ := c.AddJob("*/10 * * * * *", func() {
		fmt.Printf("[PERSISTENT] Job running at %s\n", time.Now().Format("15:04:05"))
	})

	// Check the store immediately to see history
	lastRun, _ := store.GetLastRun(id)
	if lastRun.IsZero() {
		fmt.Println("This is the first time this job has run (or store was empty).")
	} else {
		fmt.Printf("Resume detected! Job last ran at: %s\n", lastRun.Format("15:04:05"))
	}

	fmt.Println("Persistent Scheduler starting...")
	c.Start()

	// Wait generic time to see some logs
	time.Sleep(25 * time.Second)

	fmt.Println("Stopping scheduler. On next start, it will remember these runs.")
	c.Stop()
}
