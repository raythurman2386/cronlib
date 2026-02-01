# CronLib: High-Performance Go Cron Library

[![Go Reference](https://pkg.go.dev/badge/github.com/raythurman2386/cronlib.svg)](https://pkg.go.dev/github.com/raythurman2386/cronlib)
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
*   🕒 **Timezone Support**: Full support for job-specific execution locations.

## Versioning
This project follows [Semantic Versioning 2.0.0](https://semver.org/). 

We use [GoReleaser](https://goreleaser.com/) to automate our release process. Every time a new version tag (e.g., `v0.1.0`) is pushed, a corresponding GitHub Release is created with an automatically generated changelog.

## Requirements

*   **Go**: 1.24 or higher.
*   **Targeting**: Built for high-performance and modern Go concurrency patterns.

## Installation

To install the latest version:
```bash
go get github.com/raythurman2386/cronlib
```

To install a specific version:
```bash
go get github.com/raythurman2386/cronlib@v0.1.0
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

## Cron Syntax & Macros

CronLib supports standard 6-field cron syntax (sec, min, hour, dom, month, dow) and convenient macros:

*   `@yearly`, `@annually`: Run once a year.
*   `@monthly`: Run once a month.
*   `@weekly`: Run once a week.
*   `@daily`, `@midnight`: Run once a day.
*   `@hourly`: Run once an hour.

### The `@every` Syntax
For simple fixed-interval schedules, use `@every`:
```go
// Runs every 1 hour and 30 minutes
c.AddJob("@every 1h30m", func() {
    fmt.Println("Tick")
})
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

## Middleware (Job Wrappers)

You can extend job behavior using composable wrappers. CronLib applies `Recover`, `Lock` (if configured), and `Log` (if configured) by default, but you can add custom logic.

**Standard Wrappers:**
*   `Recover()`: Catches panics and prevents the scheduler from crashing.
*   `DelayIfStillRunning()`: Queues execution if the previous run hasn't finished (ensures sequential execution).
*   `SkipIfStillRunning()`: Skips execution if busy (alternative to `OverlapForbid`).

```go
c.AddJobWithOptions("@every 1s", myTask, cronlib.JobOptions{
    Wrappers: []cronlib.JobWrapper{
        cronlib.DelayIfStillRunning(),
        MyCustomMetricsWrapper(),
    },
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
