package h2h

import (
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/pools"
	"github.com/floholz/matchowl/internal/tournaments"
)

// Which matchday a pool puts first (Home's duel card, the pool page).
// While a round is open, that one. Between rounds the last result keeps
// the spotlight for a while, then the next duel takes it — at the later
// of FocusResultsFor after the last round closed (a final result always
// gets its day) and FocusNextWithin before the next round's first
// kick-off (an international break doesn't hand over weeks early).
const (
	FocusResultsFor = 24 * time.Hour
	FocusNextWithin = 72 * time.Hour
)

// nextRound is the round that opens next: the first playable one without
// a row that has not kicked off yet. Nil when the season has none left.
func nextRound(app core.App, lg *core.Record, rows []*core.Record, now time.Time) *tournaments.Round {
	have := map[string]bool{}
	for _, r := range rows {
		have[r.GetString("key")] = true
	}
	for _, r := range playable(app, lg, pools.Season(lg)) {
		if have[r.Key] || !r.First.After(now) {
			continue
		}
		return &r
	}
	return nil
}

// focusNext reports whether the next round should be put first instead
// of the last one played (rows in play order).
func focusNext(rows []*core.Record, next *tournaments.Round, now time.Time) bool {
	if next == nil {
		return false
	}
	if len(rows) == 0 {
		return true
	}
	for _, r := range rows {
		if r.GetString("status") == "open" {
			return false
		}
	}
	last := rows[len(rows)-1]
	closed := last.GetDateTime("closedAt").Time()
	if closed.IsZero() {
		closed = last.GetDateTime("closesAt").Time()
	}
	return focusFlip(closed, next.First, now)
}

// focusFlip: past the later of results-for after close and
// next-within before the next kick-off.
func focusFlip(closed, nextFirst, now time.Time) bool {
	flip := closed.Add(FocusResultsFor)
	if t := nextFirst.Add(-FocusNextWithin); t.After(flip) {
		flip = t
	}
	return !now.Before(flip)
}
