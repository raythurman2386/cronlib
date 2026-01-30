package cronlib

import (
	"testing"
)

func TestMacroParsing(t *testing.T) {
	tests := []struct {
		macro    string
		expected string
	}{
		{"@yearly", "0 0 0 1 1 *"},
		{"@annually", "0 0 0 1 1 *"},
		{"@monthly", "0 0 0 1 * *"},
		{"@weekly", "0 0 0 * * 0"},
		{"@daily", "0 0 0 * * *"},
		{"@midnight", "0 0 0 * * *"},
		{"@hourly", "0 0 * * * *"},
		// Bi-options
		{"@bi-weekly", "0 0 0 * * 0/2"},  // Every 2 weeks (DOW step)
		{"@bi-monthly", "0 0 0 1 */2 *"}, // Every 2 months
		{"@bi-daily", "0 0 0 */2 * *"},
	}

	for _, tt := range tests {
		t.Run(tt.macro, func(t *testing.T) {
			expr, err := Parse(tt.macro)
			if err != nil {
				t.Fatalf("Failed to parse macro %s: %v", tt.macro, err)
			}

			// To verify, we'll parse the expected string and compare bitmasks
			expectedExpr, _ := Parse(tt.expected)
			if expr.second != expectedExpr.second ||
				expr.minute != expectedExpr.minute ||
				expr.hour != expectedExpr.hour ||
				expr.dom != expectedExpr.dom ||
				expr.month != expectedExpr.month ||
				expr.dow != expectedExpr.dow {
				t.Errorf("Macro %s expanded incorrectly.\nGot:  %+v\nWant: %+v", tt.macro, expr, expectedExpr)
			}
		})
	}
}
