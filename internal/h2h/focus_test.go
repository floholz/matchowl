package h2h

import (
	"testing"
	"time"
)

func TestFocusFlip(t *testing.T) {
	day := 24 * time.Hour
	at := func(d float64) time.Time {
		return time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC).Add(time.Duration(d * float64(day)))
	}
	for _, tc := range []struct {
		name         string
		closed, next time.Time
		now          time.Time
		want         bool
	}{
		// Weekly: closes Mon, next Sat → hands over Wed (next − 3 days).
		{"weekly, Tue", at(0), at(5), at(1.5), false},
		{"weekly, Wed", at(0), at(5), at(2), true},
		// International break: next three weeks out → 3 days before.
		{"break, a week on", at(0), at(19), at(7), false},
		{"break, 3 days out", at(0), at(19), at(16), true},
		// Midweek: next two days out → the result still gets its day.
		{"midweek, 12h after close", at(0), at(2), at(0.5), false},
		{"midweek, a day after close", at(0), at(2), at(1), true},
	} {
		if got := focusFlip(tc.closed, tc.next, tc.now); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
