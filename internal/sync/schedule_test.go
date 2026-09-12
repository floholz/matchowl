package sync

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/football"
)

func scheduleRecord(kickoff time.Time, round string) *core.Record {
	col := core.NewBaseCollection("matches")
	col.Fields.Add(
		&core.DateField{Name: "kickoff"},
		&core.TextField{Name: "roundLabel"},
		&core.TextField{Name: "status"},
	)
	rec := core.NewRecord(col)
	rec.Set("kickoff", kickoff)
	rec.Set("roundLabel", round)
	return rec
}

func TestApplyScheduleFollowsProvider(t *testing.T) {
	placeholder := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	confirmed := time.Date(2026, 10, 4, 18, 45, 0, 0, time.UTC)
	rec := scheduleRecord(placeholder, "Regular Season - 6")

	if applySchedule(rec, football.Fixture{Date: placeholder, Round: "Regular Season - 6"}, "scheduled") {
		t.Fatal("unchanged schedule reported as changed")
	}
	if !applySchedule(rec, football.Fixture{Date: confirmed, Round: "Regular Season - 6"}, "scheduled") {
		t.Fatal("moved kick-off not applied")
	}
	if got := rec.GetDateTime("kickoff").Time().UTC(); !got.Equal(confirmed) {
		t.Fatalf("kickoff = %v, want %v", got, confirmed)
	}
	if !applySchedule(rec, football.Fixture{Date: confirmed, Round: "Regular Season - 7"}, "live") {
		t.Fatal("round label change not applied")
	}
	if rec.GetString("roundLabel") != "Regular Season - 7" {
		t.Fatalf("roundLabel = %q", rec.GetString("roundLabel"))
	}
}

func TestApplyScheduleLeavesFinishedAndUnknown(t *testing.T) {
	kickoff := time.Date(2026, 9, 12, 16, 0, 0, 0, time.UTC)
	rec := scheduleRecord(kickoff, "Regular Season - 4")
	later := kickoff.Add(48 * time.Hour)
	if applySchedule(rec, football.Fixture{Date: later, Round: "Regular Season - 4"}, "finished") {
		t.Fatal("finished match was rescheduled")
	}
	if applySchedule(rec, football.Fixture{Round: "Regular Season - 4"}, "scheduled") {
		t.Fatal("zero provider date was applied")
	}
	if got := rec.GetDateTime("kickoff").Time().UTC(); !got.Equal(kickoff) {
		t.Fatalf("kickoff moved to %v", got)
	}
}
