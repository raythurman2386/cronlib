# CronLib Feature Roadmap

This document outlines the development plan to bring `cronlib` to feature parity with established libraries like `node-cron` and `robfig/cron`.

## Phase 1: Stability & Safety (High Priority)

### 1.1 Panic Recovery
**Goal:** Ensure that a panic in a single job does not crash the entire scheduler or the application.
**Tasks:**
- [x] Implement a recovery mechanism within the job execution goroutine in `cron.go`.
- [x] Log the panic stack trace to the existing logging interface (or `stderr` if generic).
- [x] Add a unit test that intentionally panics a job and verifies the scheduler continues running.

## Phase 2: Globalization (Timezones)

### 2.1 Per-Job Timezone Support
**Goal:** Allow jobs to be scheduled in specific timezones, independent of the server's system time.
**Tasks:**
- [x] Add `Location *time.Location` field to the `Job` struct.
- [x] Add `WithLocation(loc *time.Location)` option to `JobOptions`.
- [x] Update `Expression.Next(from time.Time)` to perform calculations relative to the target timezone.
- [ ] Add support for `CRON_TZ=America/New_York ...` prefix in the spec string (optional but recommended for compatibility).
- [x] Add tests verifying daylight saving time transitions handle correctly.

## Phase 3: Enhanced Usability (Macros)

### 3.1 Standard Cron Macros
**Goal:** Support common aliases for standard schedules.
**Tasks:**
- [x] Update `Parse()` in `cron.go` to handle single-token inputs.
- [x] Implement mappings for:
    - `@yearly` / `@annually` -> `0 0 0 1 1 *`
    - `@monthly` -> `0 0 0 1 * *`
    - `@weekly` -> `0 0 0 * * 0`
    - `@daily` / `@midnight` -> `0 0 0 * * *`
    - `@hourly` -> `0 0 * * * *`
- [x] Implement bi-options (`@bi-weekly`, `@bi-monthly`, etc.)
- [x] Update tests to verify macro expansion.

### 3.2 The `@every` Syntax
**Goal:** Allow simple duration-based schedules (e.g., `@every 1h30m`).
**Tasks:**
- [x] Update `Parse()` to detect the `@every` prefix.
- [x] Parse the subsequent duration string using `time.ParseDuration`.
- [x] Create a mechanism to represent fixed-interval schedules (implemented via `Expression.interval` field).
    - *Note:* `robfig/cron` treats `@every` as a fixed delay from the start time, whereas standard cron aligns to the clock. We implemented fixed delay from start/last run.
- [x] Add tests for various duration formats.

## Phase 4: Extensibility (Middleware)

### 4.1 Job Wrappers / Interceptors
**Goal:** Refactor hardcoded logging, locking, and recovery into a composable middleware system.
**Tasks:**
- [x] Define a `JobWrapper` type (function that takes a `Job` and returns a `Job` or modifies its `Cmd`).
- [x] Implement standard wrappers:
    - `Recover()`: The logic from Phase 1.
    - `SkipIfStillRunning()`: Implemented as a wrapper (alternative to OverlapForbid).
    - `DelayIfStillRunning()`: Queuing logic.
- [x] Refactor `Cron.run` to apply these wrappers during job addition or execution, rather than having monolithic logic in the run loop.
- [x] Allow users to inject custom wrappers (e.g., for Prometheus metrics or OpenTelemetry tracing).

## Phase 5: Documentation & Cleanup
- [x] Update `README.md` with new features and examples.
- [x] Clean up `TODO` comments in the code.
- [ ] ensure `go doc` is clean and descriptive.
