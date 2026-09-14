package tournaments

import (
	"testing"
	"time"
)

func day(d int, h int) time.Time {
	return time.Date(2026, 8, d, h, 0, 0, 0, time.UTC)
}

func TestGroupRoundsOrdersByNumberNotEarliestKickoff(t *testing.T) {
	st := &Structure{Stages: []Stage{{Code: "group", Kind: KindGroup}}}
	ms := []RoundMatch{
		// Round 2 has one match pulled forward to before round 1.
		{ID: "a", Stage: "group", Label: "Regular Season - 1", Kickoff: day(15, 18)},
		{ID: "b", Stage: "group", Label: "Regular Season - 1", Kickoff: day(16, 18)},
		{ID: "c", Stage: "group", Label: "Regular Season - 2", Kickoff: day(14, 18)},
		{ID: "d", Stage: "group", Label: "Regular Season - 2", Kickoff: day(22, 18)},
		{ID: "e", Stage: "group", Label: "Regular Season - 2", Kickoff: day(23, 18)},
		{ID: "z", Stage: "group", Label: "Regular Season - 10", Kickoff: day(30, 18)},
	}
	rs := GroupRounds(st, ms)
	if len(rs) != 3 {
		t.Fatalf("want 3 rounds, got %d", len(rs))
	}
	if rs[0].Num != 1 || rs[1].Num != 2 || rs[2].Num != 10 {
		t.Fatalf("bad order: %v", []int{rs[0].Num, rs[1].Num, rs[2].Num})
	}
	if rs[1].First != day(14, 18) || rs[1].Last != day(23, 18) || rs[1].Median != day(22, 18) {
		t.Fatalf("round 2 kick-offs: first %v last %v median %v", rs[1].First, rs[1].Last, rs[1].Median)
	}
	if rs[0].Key != "group|Regular Season - 1" || len(rs[0].MatchIDs) != 2 {
		t.Fatalf("round 1: %+v", rs[0])
	}
}

func TestGroupRoundsStageOrderThenKickoff(t *testing.T) {
	st := &Structure{Stages: []Stage{
		{Code: "group", Kind: KindGroup}, {Code: "R16", Kind: KindKnockout}, {Code: "QF", Kind: KindKnockout},
	}}
	ms := []RoundMatch{
		{ID: "q", Stage: "QF", Label: "Quarter-finals", Kickoff: day(28, 18)},
		{ID: "r", Stage: "R16", Label: "Round of 16", Kickoff: day(24, 18)},
		{ID: "g2", Stage: "group", Label: "Group B", Kickoff: day(12, 18)},
		{ID: "g1", Stage: "group", Label: "Group A", Kickoff: day(11, 18)},
		{ID: "none", Stage: "group", Label: "Group A"}, // no kick-off → dropped
	}
	rs := GroupRounds(st, ms)
	got := []string{}
	for _, r := range rs {
		got = append(got, r.Label)
	}
	want := []string{"Group A", "Group B", "Round of 16", "Quarter-finals"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("order: got %v want %v", got, want)
		}
	}
	if len(rs[0].MatchIDs) != 1 {
		t.Fatalf("match without kick-off must be ignored: %+v", rs[0])
	}
}

func TestFirstRoundAfter(t *testing.T) {
	rs := []Round{{Num: 1, First: day(1, 18)}, {Num: 2, First: day(8, 18)}, {Num: 3, First: day(15, 18)}}
	if r := FirstRoundAfter(rs, day(1, 12)); r == nil || r.Num != 1 {
		t.Fatalf("before round 1: %+v", r)
	}
	if r := FirstRoundAfter(rs, day(1, 18)); r == nil || r.Num != 2 {
		t.Fatalf("at round 1's kick-off the round has started: %+v", r)
	}
	if r := FirstRoundAfter(rs, day(20, 0)); r != nil {
		t.Fatalf("season over: %+v", r)
	}
}
