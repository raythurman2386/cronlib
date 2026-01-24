# CronLib: High-Performance Go Cron Library

CronLib is a lightweight, thread-safe, and high-performance cron scheduling library for Go. It mirrors the functionality of `node-cron` but leverages Go's concurrency primitives (Goroutines, Channels, `sync.RWMutex`) to handle thousands of concurrent jobs with sub-millisecond precision.

## Key Features

*   **High Performance**: Uses a bitmask-based parser for O(1) matching of cron fields.
*   **Event-Driven Scheduler**: Avoids inefficient polling (1-second tickers). Uses a single `time.Timer` to sleep exactly until the next scheduled job.
*   **Thread Safety**: All operations (`AddJob`, `RemoveJob`, `Stop`) are safe for concurrent use.
*   **Graceful Shutdown**: Ensures all running jobs complete before the application exits.
*   **Standard Cron Syntax**: Supports 6-field cron expressions including steps (`*/5`), ranges (`1-5`), and lists (`1,3,5`).

## Installation

```bash
go get github.com/yourusername/cronlib
```

## Usage

```go
package main

import (
	"fmt"
	"time"

	"cronlib" // Replace with actual module path
)

func main() {
	c := cronlib.NewCron()

	// Add a job: runs every 5 seconds
	id, err := c.AddJob("*/5 * * * * *", func() {
		fmt.Println("Job running every 5 seconds:", time.Now())
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Added job with ID: %s\n", id)

	// Start the scheduler
	c.Start()

	// Run for 20 seconds
	time.Sleep(20 * time.Second)

	// Graceful shutdown
	fmt.Println("Stopping scheduler...")
	c.Stop()
	fmt.Println("Scheduler stopped.")
}
```

## Advanced Features

### Persistence (SQLite)
Use `cronlib/store/sqlite` to persist job states and execution logs.

```go
import "cronlib/store/sqlite"

store, _ := sqlite.New("cron.db")
c := cronlib.NewCron()
c.SetJobStore(store)
```

### Distributed Locks (Redis)
Use `cronlib/lock/redis` to prevent concurrent execution across multiple nodes.

```go
import "cronlib/lock/redis"

lock := redis.New("localhost:6379")
c := cronlib.NewCron()
c.SetDistLock(lock)
```

### Web Dashboard
A built-in dashboard to monitor jobs.

```go
import "cronlib/dashboard"

handler := dashboard.NewHandler(c)
http.ListenAndServe(":8080", handler)
```

## License

MIT
