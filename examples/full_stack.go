package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cronlib"
	"cronlib/dashboard"
	"cronlib/lock/redis"
	"cronlib/store/sqlite"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// 1. Initialize SQLite Store
	store, err := sqlite.New("cron.db")
	if err != nil {
		log.Fatalf("Failed to init store: %v", err)
	}
	defer store.Close()
	log.Println("SQLite store initialized at cron.db")

	// 2. Initialize Cron
	c := cronlib.NewCron()
	c.SetJobStore(store)

	// 3. Initialize Redis Lock (Optional)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr != "" {
		l := redis.New(redisAddr)
		// Check connection
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := l.Lock(ctx, "test-connection", time.Second); err != nil {
			log.Printf("Warning: Could not connect to Redis at %s: %v. Distributed locking disabled.", redisAddr, err)
		} else {
			c.SetDistLock(l)
			log.Printf("Redis distributed locking enabled using %s", redisAddr)
			l.Unlock(context.Background(), "test-connection")
		}
	} else {
		log.Println("Redis lock disabled (set REDIS_ADDR to enable)")
	}

	// 4. Add Jobs
	// Every 5 seconds
	id1, _ := c.AddJob("*/5 * * * * *", func() {
		log.Println("Job 1 (Every 5s) RUNNING")
		time.Sleep(500 * time.Millisecond) // Simulate work
		log.Println("Job 1 FINISHED")
	})
	fmt.Printf("Added Job 1: %s\n", id1)

	// Every 10 seconds
	id2, _ := c.AddJob("*/10 * * * * *", func() {
		log.Println("Job 2 (Every 10s) RUNNING")
	})
	fmt.Printf("Added Job 2: %s\n", id2)

	// Custom: Replace Policy
	id3, _ := c.AddJobWithOptions("*/5 * * * * *", func(ctx context.Context) {
		log.Println("Job 3 (Replace Policy) START")
		select {
		case <-time.After(4 * time.Second):
			log.Println("Job 3 FINISHED")
		case <-ctx.Done():
			log.Println("Job 3 CANCELLED (Replaced)")
		}
	}, cronlib.JobOptions{Overlap: cronlib.OverlapReplace})
	fmt.Printf("Added Job 3: %s (Replace Policy)\n", id3)

	// Start
	c.Start()
	log.Println("Cron scheduler started")

	// 5. Start Dashboard
	dashboardHandler := dashboard.NewHandler(c)
	go func() {
		log.Println("Dashboard listening on http://localhost:8080")
		if err := http.ListenAndServe(":8080", dashboardHandler); err != nil {
			log.Fatal(err)
		}
	}()

	// 6. Wait for signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nShutting down...")
	c.Stop()
	fmt.Println("Cron stopped")
}
