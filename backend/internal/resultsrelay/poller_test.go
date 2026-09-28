package resultsrelay

import (
	"testing"
	"time"
)

func TestShouldPoll(t *testing.T) {
	loc := time.UTC
	window := PollWindow{StartHour: 9, EndHour: 17}

	cases := []struct {
		name string
		when time.Time
		want bool
	}{
		{"weekday morning start", time.Date(2026, time.September, 28, 9, 0, 0, 0, loc), true},
		{"weekday afternoon", time.Date(2026, time.September, 28, 16, 59, 0, 0, loc), true},
		{"weekday before open", time.Date(2026, time.September, 28, 8, 59, 0, 0, loc), false},
		{"weekday after close", time.Date(2026, time.September, 28, 17, 0, 0, 0, loc), false},
		{"saturday during window", time.Date(2026, time.September, 26, 12, 0, 0, 0, loc), false},
		{"sunday during window", time.Date(2026, time.September, 27, 12, 0, 0, 0, loc), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldPoll(tc.when, loc, window); got != tc.want {
				t.Fatalf("ShouldPoll(%s) = %v, want %v", tc.when, got, tc.want)
			}
		})
	}
}
