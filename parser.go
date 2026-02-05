package cronlib

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Expression represents the parsed cron expression using bitmasks.
type Expression struct {
	interval time.Duration // For @every schedules
	second   uint64
	minute   uint64
	hour     uint64
	dom      uint64
	month    uint64
	dow      uint64
	// Flags for standard cron behavior
	domStar bool
	dowStar bool
}

// Parse parses a 6-field cron string or a macro into an Expression.
//
// The standard format is:
//
//	second minute hour day-of-month month day-of-week
//
// Supported macros:
//
//	@yearly, @annually (0 0 0 1 1 *)
//	@monthly           (0 0 0 1 * *)
//	@weekly            (0 0 0 * * 0)
//	@daily, @midnight  (0 0 0 * * *)
//	@hourly            (0 0 * * * *)
//	@every <duration>  (e.g., "@every 1h30m")
func Parse(spec string) (Expression, error) {
	if strings.HasPrefix(spec, "@every ") {
		durationStr := strings.TrimPrefix(spec, "@every ")
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return Expression{}, fmt.Errorf("invalid duration for @every: %w", err)
		}
		return Expression{interval: duration}, nil
	}

	if strings.HasPrefix(spec, "@") {
		switch spec {
		case "@yearly", "@annually":
			spec = "0 0 0 1 1 *"
		case "@monthly":
			spec = "0 0 0 1 * *"
		case "@bi-monthly":
			spec = "0 0 0 1 */2 *"
		case "@weekly":
			spec = "0 0 0 * * 0"
		case "@bi-weekly":
			spec = "0 0 0 * * 0/2"
		case "@daily", "@midnight":
			spec = "0 0 0 * * *"
		case "@bi-daily":
			spec = "0 0 0 */2 * *"
		case "@hourly":
			spec = "0 0 * * * *"
		}
	}

	fields := strings.Fields(spec)
	if len(fields) != 6 {
		return Expression{}, fmt.Errorf("expected 6 fields, found %d: %s", len(fields), spec)
	}

	var err error
	var expr Expression

	expr.second, _, err = parseField(fields[0], 0, 59)
	if err != nil {
		return expr, fmt.Errorf("parsing second: %w", err)
	}

	expr.minute, _, err = parseField(fields[1], 0, 59)
	if err != nil {
		return expr, fmt.Errorf("parsing minute: %w", err)
	}

	expr.hour, _, err = parseField(fields[2], 0, 23)
	if err != nil {
		return expr, fmt.Errorf("parsing hour: %w", err)
	}

	expr.dom, expr.domStar, err = parseField(fields[3], 1, 31)
	if err != nil {
		return expr, fmt.Errorf("parsing dom: %w", err)
	}

	expr.month, _, err = parseField(fields[4], 1, 12)
	if err != nil {
		return expr, fmt.Errorf("parsing month: %w", err)
	}

	expr.dow, expr.dowStar, err = parseField(fields[5], 0, 6)
	if err != nil {
		return expr, fmt.Errorf("parsing dow: %w", err)
	}

	return expr, nil
}

func parseField(field string, min, max int) (uint64, bool, error) {
	var bits uint64

	// Handle list
	parts := strings.Split(field, ",")
	for _, part := range parts {
		if part == "*" {
			for i := min; i <= max; i++ {
				bits |= (1 << i)
			}
			continue
		}

		// Handle step
		step := 1
		rangePart := part
		if i := strings.Index(part, "/"); i >= 0 {
			stepStr := part[i+1:]
			var err error
			step, err = strconv.Atoi(stepStr)
			if err != nil || step <= 0 {
				return 0, false, fmt.Errorf("invalid step: %s", part)
			}
			rangePart = part[:i]
		}

		// Handle range
		var start, end int
		if rangePart == "*" {
			start = min
			end = max
		} else if i := strings.Index(rangePart, "-"); i >= 0 {
			startStr := rangePart[:i]
			endStr := rangePart[i+1:]
			var err error
			start, err = strconv.Atoi(startStr)
			if err != nil {
				return 0, false, fmt.Errorf("invalid range start: %s", rangePart)
			}
			end, err = strconv.Atoi(endStr)
			if err != nil {
				return 0, false, fmt.Errorf("invalid range end: %s", rangePart)
			}
		} else {
			// Single number
			val, err := strconv.Atoi(rangePart)
			if err != nil {
				return 0, false, fmt.Errorf("invalid number: %s", rangePart)
			}
			start = val
			end = val
		}

		if start < min || end > max {
			return 0, false, fmt.Errorf("value out of range (%d-%d): %s", min, max, part)
		}

		for i := start; i <= end; i += step {
			if i > max {
				break
			}
			bits |= (1 << i)
		}
	}

	if bits == 0 {
		return 0, false, fmt.Errorf("no valid values in field: %s", field)
	}

	return bits, field == "*", nil
}

// Next returns the next execution time after `from`.
// Returns zero time if no match found within 5 years.
func (e Expression) Next(from time.Time) time.Time {
	// Handle fixed interval schedules (@every)
	if e.interval > 0 {
		return from.Add(e.interval)
	}

	// Start checking from the next second
	t := from.Add(1 * time.Second)
	// Strip nanoseconds
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())

	for i := 0; i < 5*366*24*60*60; i++ { // Safety limit for iterations
		if t.Year()-from.Year() > 5 {
			return time.Time{}
		}

		// 1. Month
		// #nosec G115
		if (1<<uint(t.Month()))&e.month == 0 {
			// Find next valid month
			nextMonth, ok := findNextBit(e.month, int(t.Month())+1, 1, 12)
			if !ok {
				// Next year
				nextMonth, _ = findNextBit(e.month, 1, 1, 12)
				t = time.Date(t.Year()+1, time.Month(nextMonth), 1, 0, 0, 0, 0, t.Location())
			} else {
				t = time.Date(t.Year(), time.Month(nextMonth), 1, 0, 0, 0, 0, t.Location())
			}
			continue
		}

		// 2. Day (DOM/DOW)
		// #nosec G115
		domMatch := (1<<uint(t.Day()))&e.dom != 0
		// #nosec G115
		dowMatch := (1<<uint(t.Weekday()))&e.dow != 0

		dayMatched := false
		switch {
		case !e.domStar && !e.dowStar:
			dayMatched = domMatch || dowMatch
		case e.domStar:
			dayMatched = dowMatch
		case e.dowStar:
			dayMatched = domMatch
		default:
			dayMatched = true
		}

		if !dayMatched {
			// Increment day
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}

		// 3. Hour
		// #nosec G115
		if (1<<uint(t.Hour()))&e.hour == 0 {
			nextHour, ok := findNextBit(e.hour, t.Hour()+1, 0, 23)
			if ok {
				t = time.Date(t.Year(), t.Month(), t.Day(), nextHour, 0, 0, 0, t.Location())
			} else {
				// Next day
				t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			}
			continue
		}

		// 4. Minute
		// #nosec G115
		if (1<<uint(t.Minute()))&e.minute == 0 {
			nextMinute, ok := findNextBit(e.minute, t.Minute()+1, 0, 59)
			if ok {
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), nextMinute, 0, 0, t.Location())
			} else {
				// Next hour
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			}
			continue
		}

		// 5. Second
		// #nosec G115
		if (1<<uint(t.Second()))&e.second == 0 {
			nextSecond, ok := findNextBit(e.second, t.Second()+1, 0, 59)
			if ok {
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), nextSecond, 0, t.Location())
			} else {
				// Next minute
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, t.Location())
			}
			continue
		}

		// If we are here, everything matched
		return t
	}

	return time.Time{}
}

// findNextBit returns the index of the next set bit in mask >= start.
// min/max define the range of the field (to bounds check).
func findNextBit(mask uint64, start, min, max int) (int, bool) {
	for i := start; i <= max; i++ {
		if (mask & (1 << i)) != 0 {
			return i, true
		}
	}
	return 0, false
}
