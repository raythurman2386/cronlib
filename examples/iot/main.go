package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/raythurman2386/cronlib"
)

func main() {
	c := cronlib.NewCron()

	// High-frequency sensor polling (every 2 seconds)
	// Demonstrates O(1) matching and sub-millisecond precision
	c.AddJob("*/2 * * * * *", func() {
		now := time.Now()
		// Calculate offset from exact second boundary to demonstrate precision
		offset := now.Sub(now.Truncate(2 * time.Second))

		temp := 20.0 + rand.Float64()*10.0
		fmt.Printf("[Precision Trace] Sensor 1 read at %s (offset: %v) -> Value: %.2f°C\n",
			now.Format("15:04:05.000"), offset, temp)
	})

	// Heavy duty analytics (every 10 seconds)
	c.AddJob("*/10 * * * * *", func() {
		fmt.Printf("[Analytics] Aggregating sensor data streams at %s...\n",
			time.Now().Format("15:04:05"))
		time.Sleep(500 * time.Millisecond) // Simulate aggregation logic
		fmt.Println("[Analytics] Write to time-series DB successful.")
	})

	fmt.Println("IoT Ingestion Engine starting...")
	fmt.Println("Running for 30 seconds. Watch the precision offsets (should be <10ms).")

	c.Start()
	time.Sleep(30 * time.Second)
	c.Stop()
	fmt.Println("Ingestion Engine stopped.")
}
