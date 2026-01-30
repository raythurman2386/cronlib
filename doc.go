/*
Package cronlib provides a high-performance, thread-safe cron job scheduler for Go.

It allows scheduling jobs using standard cron syntax with seconds precision, as well
as convenient macros like @every, @daily, etc.

# Features

  - **Thread-Safe**: Safe for concurrent use from multiple goroutines.
  - **High Performance**: Uses a bitmask-based parser for fast matching.
  - **Seconds Precision**: Supports 6-field cron expressions (second, minute, hour, dom, month, dow).
  - **Job Overlap Policies**: Control behavior when a previous job run is still active (Allow, Forbid, Replace).
  - **Middleware/Wrappers**: Extensible job execution pipeline (logging, recovery, locking).
  - **Persistence**: Interfaces for storing job state and execution history.
  - **Distributed Locking**: Interfaces for cluster-wide job synchronization.

# Syntax

The cron expression format is a space-separated string with 6 fields:

	Field name   | Mandatory? | Allowed values  | Allowed special characters
	----------   | ---------- | --------------  | --------------------------
	Seconds      | Yes        | 0-59            | * / , -
	Minutes      | Yes        | 0-59            | * / , -
	Hours        | Yes        | 0-23            | * / , -
	Day of month | Yes        | 1-31            | * / , -
	Month        | Yes        | 1-12            | * / , -
	Day of week  | Yes        | 0-6 (Sun-Sat)   | * / , -

# Macros

Pre-defined macros can be used in place of the cron expression:

	@yearly (or @annually)  Run once a year, midnight, Jan. 1st        (0 0 0 1 1 *)
	@monthly                Run once a month, midnight, first of month  (0 0 0 1 * *)
	@weekly                 Run once a week, midnight between Sat/Sun   (0 0 0 * * 0)
	@daily (or @midnight)   Run once a day, midnight                    (0 0 0 * * *)
	@hourly                 Run once an hour, beginning of hour         (0 0 * * * *)
	@every <duration>       Run every <duration> (e.g. "@every 1h30m")

# Usage Example

	package main

	import (
		"fmt"
		"time"
		"github.com/raythurman2386/cronlib"
	)

	func main() {
		c := cronlib.NewCron()

		// Run every 5 seconds
		c.AddJob("*\0575 * * * * *", func() {
			fmt.Println("Tick every 5s")
		})

		// Run every minute
		c.AddJob("@every 1m", func() {
			fmt.Println("Tick every 1m")
		})

		c.Start()
		defer c.Stop()

		// Block main thread
		select {}
	}
*/
package cronlib
