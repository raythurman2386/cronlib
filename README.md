# CronLib: High-Performance Go Cron Library

[![Go Reference](https://pkg.go.dev/badge/github.com/raythurman2386/cronlib.svg)](https://pkg.go.dev/github.com/raythurman2386/cronlib)
[![Go Report Card](https://goreportcard.com/badge/github.com/raythurman2386/cronlib)](https://goreportcard.com/report/github.com/raythurman2386/cronlib)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**CronLib** is a lightweight, thread-safe, and high-performance cron scheduling library for Go. It is designed to handle thousands of concurrent jobs with sub-millisecond precision, mirroring `node-cron` functionality but optimized for the Go ecosystem.

## Key Features

*   🚀 **High Performance**: Bitmask-based parser for O(1) matching of cron fields.
*   🕒 **Sub-millisecond Precision**: Uses `time.Timer` for event-driven scheduling (no polling tickers).
*   🔒 **Thread-Safe**: Safe for concurrent job management (adding, removing, stopping).
*   ⚙️ **Overlap Policies**: Control job execution overlap with `Allow`, `Forbid`, or `Replace` policies.
*   💾 **Persistence**: Native SQLite support to track last run times and execution history.
*   🌐 **Distributed Locks**: Redis integration for cluster-wide job synchronization.
*   🖥️ **Web Dashboard**: Built-in UI to monitor job status and execution logs in real-time.

## Installation

```bash
go get github.com/raythurman2386/cronlib
```

## Quick Start

```go
package main

import (
	"fmt"
	"time"
	"github.com/raythurman2386/cronlib"
)

func main() {
	c := cronlib.NewCron()

	// Add a job: runs every 5 seconds
	c.AddJob("*/5 * * * * *", func() {
		fmt.Println("Tick:", time.Now().Format("15:04:05"))
	})

	c.Start()
	select {} // Keep running
}
```

## Concurrency Control (Overlap Policies)

CronLib provides fine-grained control over how jobs behave when a new execution is scheduled while a previous instance is still running.

| Policy | Description |
| :--- | :--- |
| `OverlapAllow` | (Default) Allows multiple instances to run concurrently. |
| `OverlapForbid` | Skips the execution if the previous instance is still running. |
| `OverlapReplace` | Cancels the running instance and starts the new one immediately. |

```go
c.AddJobWithOptions("*/10 * * * * *", myTask, cronlib.JobOptions{
    Overlap: cronlib.OverlapReplace,
})
```

## Advanced Production Features

### 1. Persistent State (SQLite)
Track job history and recover schedules across restarts.

```go
import "github.com/raythurman2386/cronlib/store/sqlite"

store, _ := sqlite.New("cron.db")
c.SetJobStore(store)
```

### 2. Distributed Locks (Redis)
Ensure a job runs only once across a cluster.

```go
import "github.com/raythurman2386/cronlib/lock/redis"

lock := redis.New("localhost:6379")
c.SetDistLock(lock)
```

### 3. Web Dashboard
Embedded UI accessible at `http://localhost:8080`.

```go
import "github.com/raythurman2386/cronlib/dashboard"

http.Handle("/", dashboard.NewHandler(c))
http.ListenAndServe(":8080", nil)
```

## Monitoring
The Web Dashboard provides a live view of:
- **Job ID & Expression**
- **Next Scheduled Run**
- **Last Execution Time**
- **Real-time Status** (Running/Idle)

## Examples

Explore more realistic implementation patterns in the `examples/` directory:

- 🚀 **[IoT Ingestion](./examples/iot/main.go)**: High-frequency polling with sub-millisecond precision monitoring.
- 🔒 **[Distributed Singleton](./examples/distributed/main.go)**: Cluster-wide job synchronization using Redis locks.
- 💾 **[Persistent Recovery](./examples/persistence/main.go)**: Resuming schedules and tracking history using SQLite.
- ⚙️ **[Overlap Control](./examples/overlap/main.go)**: Demonstrating `Forbid` and `Replace` policies for slow tasks.
- 🌐 **[Full Stack](./examples/fullstack/main.go)**: A complete implementation featuring the dashboard, persistence, and locking.

You can run any example using:
```bash
go run examples/iot/main.go
```

## License

MIT
