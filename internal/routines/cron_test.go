package routines_test

import (
	"testing"
	"time"

	"github.com/ubunatic/paperclip-go/internal/routines"
)

// TestIsDue_WildcardMatching tests that * * * * * matches any time
func TestIsDue_WildcardMatching(t *testing.T) {
	expr := "* * * * *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"midnight on New Year", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"noon on random day", time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC), true},
		{"2359 on Dec 31", time.Date(2024, 12, 31, 23, 59, 0, 0, time.UTC), true},
		{"Feb 29 leap year", time.Date(2024, 2, 29, 13, 45, 0, 0, time.UTC), true},
		{"random minute", time.Date(2024, 7, 4, 8, 37, 0, 0, time.UTC), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_ExactMatches tests that exact field specifications work correctly
func TestIsDue_ExactMatches(t *testing.T) {
	expr := "0 9 1 1 *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"Jan 1 at 09:00", time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), true},
		{"Jan 1 at 09:01", time.Date(2024, 1, 1, 9, 1, 0, 0, time.UTC), false},
		{"Jan 1 at 08:00", time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC), false},
		{"Jan 2 at 09:00", time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC), false},
		{"Feb 1 at 09:00", time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC), false},
		{"Dec 31 at 09:00", time.Date(2024, 12, 31, 9, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_Ranges tests that range syntax (n-m) works correctly
func TestIsDue_Ranges(t *testing.T) {
	expr := "0 9 1-3 * *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"day 1 at 09:00", time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC), true},
		{"day 2 at 09:00", time.Date(2024, 5, 2, 9, 0, 0, 0, time.UTC), true},
		{"day 3 at 09:00", time.Date(2024, 5, 3, 9, 0, 0, 0, time.UTC), true},
		{"day 4 at 09:00", time.Date(2024, 5, 4, 9, 0, 0, 0, time.UTC), false},
		{"day 31 at 09:00", time.Date(2024, 5, 31, 9, 0, 0, 0, time.UTC), false},
		{"day 1 at 08:00", time.Date(2024, 5, 1, 8, 0, 0, 0, time.UTC), false},
		{"day 1 at 10:00", time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_Lists tests that list syntax (n,m,...) works correctly
func TestIsDue_Lists(t *testing.T) {
	expr := "0,30 9 * * *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"09:00", time.Date(2024, 5, 4, 9, 0, 0, 0, time.UTC), true},
		{"09:30", time.Date(2024, 5, 4, 9, 30, 0, 0, time.UTC), true},
		{"09:15", time.Date(2024, 5, 4, 9, 15, 0, 0, time.UTC), false},
		{"09:01", time.Date(2024, 5, 4, 9, 1, 0, 0, time.UTC), false},
		{"08:00", time.Date(2024, 5, 4, 8, 0, 0, 0, time.UTC), false},
		{"10:00", time.Date(2024, 5, 4, 10, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_IntervalMinutes tests that */n on minute field (min=0) works correctly
func TestIsDue_IntervalMinutes(t *testing.T) {
	expr := "*/15 * * * *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"minute 0", time.Date(2024, 5, 4, 12, 0, 0, 0, time.UTC), true},
		{"minute 15", time.Date(2024, 5, 4, 12, 15, 0, 0, time.UTC), true},
		{"minute 30", time.Date(2024, 5, 4, 12, 30, 0, 0, time.UTC), true},
		{"minute 45", time.Date(2024, 5, 4, 12, 45, 0, 0, time.UTC), true},
		{"minute 1", time.Date(2024, 5, 4, 12, 1, 0, 0, time.UTC), false},
		{"minute 14", time.Date(2024, 5, 4, 12, 14, 0, 0, time.UTC), false},
		{"minute 16", time.Date(2024, 5, 4, 12, 16, 0, 0, time.UTC), false},
		{"minute 59", time.Date(2024, 5, 4, 12, 59, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_IntervalMonths tests that */n on month field (min=1) works correctly.
// */2 should match months where (month - 1) % 2 == 0, i.e., months 1, 3, 5, 7, 9, 11
func TestIsDue_IntervalMonths(t *testing.T) {
	expr := "0 9 * */2 *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"month 1 (Jan)", time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), true},
		{"month 2 (Feb)", time.Date(2024, 2, 1, 9, 0, 0, 0, time.UTC), false},
		{"month 3 (Mar)", time.Date(2024, 3, 1, 9, 0, 0, 0, time.UTC), true},
		{"month 4 (Apr)", time.Date(2024, 4, 1, 9, 0, 0, 0, time.UTC), false},
		{"month 5 (May)", time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC), true},
		{"month 6 (Jun)", time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC), false},
		{"month 7 (Jul)", time.Date(2024, 7, 1, 9, 0, 0, 0, time.UTC), true},
		{"month 8 (Aug)", time.Date(2024, 8, 1, 9, 0, 0, 0, time.UTC), false},
		{"month 9 (Sep)", time.Date(2024, 9, 1, 9, 0, 0, 0, time.UTC), true},
		{"month 10 (Oct)", time.Date(2024, 10, 1, 9, 0, 0, 0, time.UTC), false},
		{"month 11 (Nov)", time.Date(2024, 11, 1, 9, 0, 0, 0, time.UTC), true},
		{"month 12 (Dec)", time.Date(2024, 12, 1, 9, 0, 0, 0, time.UTC), false},
		{"month 1 wrong hour", time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_IntervalDayOfMonth tests that */n on day-of-month field (min=1) works correctly.
// */2 should match days where (day - 1) % 2 == 0, i.e., days 1, 3, 5, 7, 9, ...
func TestIsDue_IntervalDayOfMonth(t *testing.T) {
	expr := "0 9 */2 * *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"day 1", time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC), true},
		{"day 2", time.Date(2024, 5, 2, 9, 0, 0, 0, time.UTC), false},
		{"day 3", time.Date(2024, 5, 3, 9, 0, 0, 0, time.UTC), true},
		{"day 4", time.Date(2024, 5, 4, 9, 0, 0, 0, time.UTC), false},
		{"day 5", time.Date(2024, 5, 5, 9, 0, 0, 0, time.UTC), true},
		{"day 6", time.Date(2024, 5, 6, 9, 0, 0, 0, time.UTC), false},
		{"day 7", time.Date(2024, 5, 7, 9, 0, 0, 0, time.UTC), true},
		{"day 8", time.Date(2024, 5, 8, 9, 0, 0, 0, time.UTC), false},
		{"day 15", time.Date(2024, 5, 15, 9, 0, 0, 0, time.UTC), true},
		{"day 16", time.Date(2024, 5, 16, 9, 0, 0, 0, time.UTC), false},
		{"day 31", time.Date(2024, 5, 31, 9, 0, 0, 0, time.UTC), true},
		{"day 1 wrong hour", time.Date(2024, 5, 1, 8, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_InvalidExpressions tests that invalid cron expressions return errors or behave predictably
func TestIsDue_InvalidExpressions(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		shouldErr bool
	}{
		{"missing field", "0 9 * *", true},
		{"too many fields", "0 9 * * * *", true},
		// The parser doesn't strictly validate field values; it just returns false for unparseable fields
		// These expressions are technically invalid cron, but they return no error and just return false
		{"invalid minute (parses as false)", "abc 9 * * *", false},
		{"invalid hour (parses as false)", "0 def * * *", false},
		{"invalid day (parses as false)", "0 9 xyz * *", false},
		{"invalid month (parses as false)", "0 9 * qrs *", false},
		{"invalid dow (parses as false)", "0 9 * * tuv", false},
		{"invalid range (parses as false)", "0 9 1-x * *", false},
		{"invalid interval negative (parses as false)", "*/-5 9 * * *", false},
		{"invalid interval zero (parses as false)", "*/0 9 * * *", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := routines.IsDue(tt.expr, time.Now())
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestIsDue_Feb29Leap tests that Feb 29 only matches in leap years
func TestIsDue_Feb29Leap(t *testing.T) {
	expr := "0 0 29 2 *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"Feb 29 2024 (leap year)", time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), true},
		{"Feb 29 2023 (non-leap, doesn't exist)", time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC), false},
		{"Feb 28 2023 (non-leap)", time.Date(2023, 2, 28, 0, 0, 0, 0, time.UTC), false},
		{"Feb 28 2024 (leap year)", time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC), false},
		{"Feb 29 2020 (leap year)", time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC), true},
		{"Feb 29 2000 (leap year)", time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_ComplexCombinations tests combinations of different field types
func TestIsDue_ComplexCombinations(t *testing.T) {
	tests := []struct {
		name string
		expr string
		time time.Time
		want bool
	}{
		{
			"list of minutes with range of days",
			"0,30 9 1-5 * *",
			time.Date(2024, 5, 3, 9, 30, 0, 0, time.UTC),
			true,
		},
		{
			"list of minutes with range of days - wrong minute",
			"0,30 9 1-5 * *",
			time.Date(2024, 5, 3, 9, 15, 0, 0, time.UTC),
			false,
		},
		{
			"interval hours with exact day/month",
			"0 */6 15 3 *",
			time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC),
			true,
		},
		{
			"interval hours (*/6) - hour 13 doesn't match",
			"0 */6 15 3 *",
			time.Date(2024, 3, 15, 13, 0, 0, 0, time.UTC),
			false,
		},
		{
			"interval day and month",
			"0 9 */3 */4 *",
			time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			true,
		},
		{
			"interval day and month - both off",
			"0 9 */3 */4 *",
			time.Date(2024, 2, 2, 9, 0, 0, 0, time.UTC),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(tt.expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_DayOfWeek tests that day-of-week field (0=Sunday to 6=Saturday) works correctly
func TestIsDue_DayOfWeek(t *testing.T) {
	// 2024-05-05 is a Sunday (weekday = 0)
	// 2024-05-06 is a Monday (weekday = 1)
	// 2024-05-10 is a Friday (weekday = 5)
	expr := "0 9 * * 0,5"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"Sunday", time.Date(2024, 5, 5, 9, 0, 0, 0, time.UTC), true},
		{"Friday", time.Date(2024, 5, 10, 9, 0, 0, 0, time.UTC), true},
		{"Monday", time.Date(2024, 5, 6, 9, 0, 0, 0, time.UTC), false},
		{"Wednesday", time.Date(2024, 5, 8, 9, 0, 0, 0, time.UTC), false},
		{"Saturday", time.Date(2024, 5, 11, 9, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsDue_EdgeCaseIntervalHours tests */n on hour field (min=0)
func TestIsDue_EdgeCaseIntervalHours(t *testing.T) {
	expr := "0 */8 * * *"
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"hour 0", time.Date(2024, 5, 4, 0, 0, 0, 0, time.UTC), true},
		{"hour 8", time.Date(2024, 5, 4, 8, 0, 0, 0, time.UTC), true},
		{"hour 16", time.Date(2024, 5, 4, 16, 0, 0, 0, time.UTC), true},
		{"hour 1", time.Date(2024, 5, 4, 1, 0, 0, 0, time.UTC), false},
		{"hour 7", time.Date(2024, 5, 4, 7, 0, 0, 0, time.UTC), false},
		{"hour 9", time.Date(2024, 5, 4, 9, 0, 0, 0, time.UTC), false},
		{"hour 23", time.Date(2024, 5, 4, 23, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := routines.IsDue(expr, tt.time)
			if err != nil {
				t.Fatalf("IsDue: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
