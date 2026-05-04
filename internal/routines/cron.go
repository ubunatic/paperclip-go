package routines

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// IsDue reports whether a 5-field cron expression matches the given time (truncated to minute).
func IsDue(expr string, t time.Time) (bool, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false, fmt.Errorf("invalid cron expression: expected 5 fields, got %d", len(fields))
	}

	// Truncate to minute
	t = t.Truncate(time.Minute)

	minute := t.Minute()
	hour := t.Hour()
	dom := t.Day()
	month := int(t.Month())
	dow := int(t.Weekday())

	// Parse and check each field
	if !matchesField(fields[0], minute, 0, 59) {
		return false, nil
	}
	if !matchesField(fields[1], hour, 0, 23) {
		return false, nil
	}
	if !matchesField(fields[2], dom, 1, 31) {
		return false, nil
	}
	if !matchesField(fields[3], month, 1, 12) {
		return false, nil
	}
	if !matchesField(fields[4], dow, 0, 6) {
		return false, nil
	}

	return true, nil
}

// matchesField checks if value matches a cron field specification.
// Supports: *, n, n-m, */n, n,m,...
func matchesField(field string, value, min, max int) bool {
	if field == "*" {
		return true
	}

	// Check for list (comma-separated)
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		for _, part := range parts {
			if matchesField(part, value, min, max) {
				return true
			}
		}
		return false
	}

	// Check for interval (*/n)
	if strings.HasPrefix(field, "*/") {
		interval := field[2:]
		n, err := strconv.Atoi(interval)
		if err != nil || n <= 0 {
			return false
		}
		return value%n == 0 || (min > 0 && (value-min)%n == 0)
	}

	// Check for range (n-m)
	if strings.Contains(field, "-") {
		parts := strings.Split(field, "-")
		if len(parts) != 2 {
			return false
		}
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return false
		}
		return value >= start && value <= end
	}

	// Single value (n)
	n, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return value == n
}

// NextAfter returns the next time after t that the cron expression fires.
// Advances minute-by-minute until IsDue returns true.
func NextAfter(expr string, t time.Time) (time.Time, error) {
	// Truncate to start of next minute
	next := t.Truncate(time.Minute).Add(time.Minute)

	// Search up to 4 years ahead to find next match
	for i := 0; i < 4*365*24*60; i++ {
		due, err := IsDue(expr, next)
		if err != nil {
			return time.Time{}, err
		}
		if due {
			return next, nil
		}
		next = next.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("no matching time found within 4 years")
}
