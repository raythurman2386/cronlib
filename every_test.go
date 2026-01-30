package cronlib

import (
	"sync"
	"testing"
	"time"
)

func TestParseEvery(t *testing.T) {
	tests := []struct {
		spec        string
		expectError bool
		expectDur   time.Duration
	}{
		{"@every 1h", false, time.Hour},
		{"@every 1m30s", false, 90 * time.Second},
		{"@every 500ms", false, 500 * time.Millisecond},
		{"@every", true, 0},
		{"@every invalid", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			expr, err := Parse(tt.spec)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for spec %q, got nil", tt.spec)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for spec %q: %v", tt.spec, err)
				}
				if expr.interval != tt.expectDur {
					t.Errorf("Expected interval %v, got %v", tt.expectDur, expr.interval)
				}
			}
		})
	}
}

func TestEveryNext(t *testing.T) {
	spec := "@every 1h"
	expr, err := Parse(spec)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	start := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	next := expr.Next(start)
	expected := start.Add(time.Hour)

	if !next.Equal(expected) {
		t.Errorf("Expected next time %v, got %v", expected, next)
	}
}

func TestEveryIntegration(t *testing.T) {
	c := NewCron()
	var mu sync.Mutex
	count := 0

	// Run frequently to ensure we catch multiple runs quickly
	_, err := c.AddJob("@every 100ms", func() {
		mu.Lock()
		count++
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("Failed to add job: %v", err)
	}

	c.Start()
	defer c.Stop()

	// Wait enough time for at least 3 runs (300ms + buffer)
	time.Sleep(450 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count < 3 {
		t.Errorf("Expected at least 3 runs, got %d", count)
	}
}
