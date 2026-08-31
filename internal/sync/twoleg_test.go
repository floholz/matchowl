package sync

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func legCol() *core.Collection {
	col := core.NewBaseCollection("matches")
	col.Fields.Add(&core.TextField{Name: "status"})
	col.Fields.Add(&core.TextField{Name: "homeTeam"})
	col.Fields.Add(&core.TextField{Name: "awayTeam"})
	for _, f := range []string{"ftHome", "ftAway", "etHome", "etAway", "penHome", "penAway"} {
		col.Fields.Add(&core.NumberField{Name: f, OnlyInt: true})
	}
	return col
}

func leg(col *core.Collection, home, away string, ftH, ftA int) *core.Record {
	m := core.NewRecord(col)
	m.Set("status", "finished")
	m.Set("homeTeam", home)
	m.Set("awayTeam", away)
	m.Set("ftHome", ftH)
	m.Set("ftAway", ftA)
	return m
}

func TestTieAdvancer(t *testing.T) {
	col := legCol()

	// The motivating case: A wins leg 1 big, loses leg 2 narrowly — A still
	// advances on aggregate even though B won the deciding leg.
	l1 := leg(col, "A", "B", 3, 0)
	l2 := leg(col, "B", "A", 1, 0)
	if got := tieAdvancer(l1, l2); got != "A" {
		t.Fatalf("aggregate winner = %q, want A", got)
	}

	// Level after 90' of leg 2 → extra time (et = cumulative after-120).
	l1 = leg(col, "A", "B", 1, 0)
	l2 = leg(col, "B", "A", 1, 0)
	l2.Set("etHome", 2)
	l2.Set("etAway", 0)
	if got := tieAdvancer(l1, l2); got != "B" {
		t.Fatalf("ET aggregate winner = %q, want B", got)
	}

	// Still level after 120' → leg 2 shootout decides.
	l1 = leg(col, "A", "B", 1, 1)
	l2 = leg(col, "B", "A", 2, 2)
	l2.Set("etHome", 2)
	l2.Set("etAway", 2)
	l2.Set("penHome", 4)
	l2.Set("penAway", 3)
	if got := tieAdvancer(l1, l2); got != "B" {
		t.Fatalf("shootout winner = %q, want B", got)
	}

	// Undecidable: aggregate level, no shootout recorded yet.
	l2.Set("penHome", 0)
	l2.Set("penAway", 0)
	if got := tieAdvancer(l1, l2); got != "" {
		t.Fatalf("undecided tie advancer = %q, want empty", got)
	}

	// Leg 2 not finished → no advancer yet.
	l1 = leg(col, "A", "B", 3, 0)
	l2 = leg(col, "B", "A", 1, 0)
	l2.Set("status", "live")
	if got := tieAdvancer(l1, l2); got != "" {
		t.Fatalf("unfinished tie advancer = %q, want empty", got)
	}

	// Cup replay: leg 1 drawn, rematch decided — rematch winner advances.
	l1 = leg(col, "A", "B", 2, 2)
	l2 = leg(col, "B", "A", 1, 0)
	if got := tieAdvancer(l1, l2); got != "B" {
		t.Fatalf("replay winner = %q, want B", got)
	}
}
