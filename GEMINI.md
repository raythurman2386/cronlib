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

*   **`cron.go`**: The core of the library. Contains:
    *   `Cron`: The main scheduler engine managing the job list and run loop.
    *   `Job`: Represents a scheduled task with its expression and state.
    *   `Expression`: The parsed cron schedule using bitmasks.
    *   `Parse()`: Logic to convert cron strings into `Expression` objects.
    *   `run()`: The main event loop that waits for the next job or signal.
*   **`*_test.go`**: Unit tests ensuring correctness of parsing, scheduling, and concurrency.

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
*   **No External Dependencies:** The project relies solely on the Go standard library.
*   **Formatting:** Standard `gofmt` style.

## Future Roadmap (from README)
*   Distributed locks (Redis/Etcd integration).
*   Persistence layer (SQLite/BadgerDB).
*   Web Dashboard for monitoring.
