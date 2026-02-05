package main

import (
	"fmt"
	"os"
	"time"

	"github.com/raythurman2386/cronlib"
	"github.com/raythurman2386/cronlib/pkg/lock/redis"
)

func main() {
	c := cronlib.NewCron()

	// Configuration
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// 1. Initialize Redis Lock
	// In a real cluster environment, multiple instances of this binary would be running.
	// The Redis lock ensures that regardless of how many instances exist, only one
	// executes the "Singleton" tasks.
	lock := redis.New(redisAddr)
	c.SetDistLock(lock)

	// 2. Add a Singleton Job (Every 5 seconds)
	// By prefixing the lock key (handled internally as "cron:1"),
	// nodes compete for this execution.
	c.AddJob("*/5 * * * * *", func() {
		fmt.Printf("[NODE AT %s] Acquired distributed lock! Processing Daily Reports...\n",
			time.Now().Format("15:04:05"))
		time.Sleep(2 * time.Second) // Simulate work
		fmt.Println("[REPORT] Reports generated successfully.")
	})

	fmt.Printf("Cluster Node starting (Redis: %s)...\n", redisAddr)
	fmt.Println("If you run multiple instances of this example, only one will process the report.")

	c.Start()

	// Keep running until signal
	select {
	case <-time.After(30 * time.Second):
		fmt.Println("Example finished after 30s")
	}

	c.Stop()
}
