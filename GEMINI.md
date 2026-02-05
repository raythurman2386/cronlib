# CronLib

**CronLib** is a lightweight, thread-safe, and high-performance cron scheduling library for Go. It is designed to handle thousands of concurrent jobs with sub-millisecond precision using Go's concurrency primitives.

## Project Overview

*   **Purpose:** A robust job scheduler for Go applications, mirroring `node-cron` functionality but optimized for the Go ecosystem.
*   **Key Features:**
    *   **Bitmask-based Parser:** O(1) matching for cron fields.
    *   **Event-Driven:** Uses `time.Timer` instead of polling tickers.
    *   **Thread-Safe:** Safe for concurrent job management.
    *   **Overlap Policies:** Controls behavior when a job is triggered while a previous instance is running (`Allow`, `Forbid`, `Replace`).
    *   **Standard Syntax:** Supports 6-field cron expressions (sec, min, hour, dom, month, dow).

## Key Files & Architecture

*   **`cron.go`**: The core scheduler engine (`Cron` struct).
*   **`parser.go`**: Logic for parsing cron expressions and calculating next run times (`Expression`, `Parse`).
*   **`job.go`**: Job-related data structures (`Job`, `JobOptions`, `JobStatus`).
*   **`interfaces.go`**: Core interfaces (`JobStore`, `DistLock`) and policy definitions.
*   **`pkg/`**: Auxiliary packages and extensions:
    *   `pkg/dashboard`: Web UI for monitoring.
    *   `pkg/store/sqlite`: Persistence implementation.
    *   `pkg/lock/redis`: Distributed locking implementation.
*   **`*_test.go`**: Comprehensive unit and integration tests.

## Building and Running

This is a standard Go module.

### Build
To build the project:
```bash
go build ./...
```

### Test
To run the test suite:
```bash
go test ./... -v
```

## Development Conventions

*   **Language:** Go (1.24+)
*   **Concurrency:** Heavy reliance on Goroutines, Channels (`stopCh`, `addCh`), `sync.RWMutex` for state protection, and `sync.WaitGroup` for graceful shutdowns.
*   **Context:** Uses `context.Context` for job cancellation, especially relevant for the `OverlapReplace` policy.
*   **No External Dependencies:** The core library relies solely on the Go standard library. External implementations (SQLite, Redis) are in `pkg/`.
*   **Formatting:** Standard `gofmt` style.
