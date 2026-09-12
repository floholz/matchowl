package sync

import (
	"testing"

	"github.com/floholz/matchowl/internal/importer"
	"github.com/floholz/matchowl/internal/tournaments"
)

// A UCL import today has only the league phase; the knockout stages arrive
// one draw at a time via backfillMatch → appendStage.
func TestAppendStageUCLDraws(t *testing.T) {
	st := &tournaments.Structure{
		Stages:    []tournaments.Stage{{Code: "group", Name: "Pool", Kind: tournaments.KindGroup}},
		GroupSize: 36, GamesPerTeam: 8,
	}
	st.Normalize()
	for _, round := range []string{"Knockout Round Play-offs", "Round of 16", "Quarter-finals", "Semi-finals", "Final"} {
		changed, err := appendStage(st, importer.StageFor(round))
		if err != nil || !changed {
			t.Fatalf("append %q: changed=%v err=%v", round, changed, err)
		}
	}
	want := []string{"group", "KOPO", "R16", "QF", "SF", "FINAL"}
	got := st.StageCodes()
	if len(got) != len(want) {
		t.Fatalf("stages = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stages = %v, want %v", got, want)
		}
	}
	// Later sync runs: the stage exists, nothing changes.
	if changed, err := appendStage(st, importer.StageFor("Round of 16")); changed || err != nil {
		t.Fatalf("re-append: changed=%v err=%v", changed, err)
	}
	// A league-phase fixture maps onto the existing group stage.
	if changed, err := appendStage(st, importer.StageFor("Pool Stage - 9")); changed || err != nil {
		t.Fatalf("pool round: changed=%v err=%v", changed, err)
	}
	if err := st.Validate(); err != nil {
		t.Fatalf("structure invalid: %v", err)
	}
}

func TestAppendStageRefusesGroupWithoutGroupStage(t *testing.T) {
	st := &tournaments.Structure{
		Stages: []tournaments.Stage{{Code: "FINAL", Name: "Final", Kind: tournaments.KindKnockout}},
	}
	if changed, err := appendStage(st, importer.StageFor("Regular Season - 1")); err == nil || changed {
		t.Fatalf("expected refusal, got changed=%v err=%v", changed, err)
	}
}
