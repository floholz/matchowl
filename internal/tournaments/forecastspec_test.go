package tournaments

import (
	"sort"
	"testing"
)

func sortedCopy(m map[string][]string, k string) []string {
	out := append([]string(nil), m[k]...)
	sort.Strings(out)
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRelationsLeague(t *testing.T) {
	st := &Structure{
		Stages:    []Stage{{Code: "group", Name: "League", Kind: KindGroup}},
		GroupSize: 20, GamesPerTeam: 38,
		Zones: []Zone{
			{Key: "ucl", Name: "Champions League", From: 1, To: 4},
			{Key: "top6", Name: "Europe", From: 1, To: 6},
			{Key: "rel", Name: "Relegated", From: 18, To: 20},
		},
	}
	st.Normalize()
	spec := &ForecastSpec{Mode: ForecastCalls, Calls: []Call{
		{Key: "winner", Name: "Winner", Type: CallTeam, Points: 100},
		{Key: "ucl", Name: "UCL", Type: CallTeamset, Points: 60, Zone: "ucl"},
		{Key: "top6", Name: "Europe", Type: CallTeamset, Points: 40, Zone: "top6"},
		{Key: "rel", Name: "Relegated", Type: CallTeamset, Points: 75, Zone: "rel"},
	}}
	if err := spec.Validate(st); err != nil {
		t.Fatal(err)
	}
	implies, exclusive := spec.Relations(st)
	if got := sortedCopy(implies, "winner"); !eq(got, []string{"top6", "ucl"}) {
		t.Fatalf("winner implies %v", got)
	}
	if got := sortedCopy(implies, "ucl"); !eq(got, []string{"top6"}) {
		t.Fatalf("ucl implies %v", got)
	}
	if len(implies["top6"]) != 0 || len(implies["rel"]) != 0 {
		t.Fatalf("unexpected implies: %v", implies)
	}
	if got := sortedCopy(exclusive, "rel"); !eq(got, []string{"top6", "ucl", "winner"}) {
		t.Fatalf("rel exclusive %v", got)
	}
	if got := sortedCopy(exclusive, "winner"); !eq(got, []string{"rel"}) {
		t.Fatalf("winner exclusive %v", got)
	}
}

func TestRelationsKnockout(t *testing.T) {
	st := &Structure{Stages: []Stage{
		{Code: "group", Name: "Groups", Kind: KindGroup},
		{Code: "QF", Name: "Quarter-finals", Kind: KindKnockout},
		{Code: "SF", Name: "Semi-finals", Kind: KindKnockout},
		{Code: "3RD", Name: "Third place", Kind: KindKnockout, Consolation: true},
		{Code: "FINAL", Name: "Final", Kind: KindKnockout},
	}, GroupSize: 4, GamesPerTeam: 3, DirectQualifiers: 2}
	st.Normalize()
	spec := &ForecastSpec{Mode: ForecastCalls, Calls: []Call{
		{Key: "champ", Name: "Champion", Type: CallTeam, Points: 50},
		{Key: "final", Name: "Finalists", Type: CallTeamset, Points: 20, Stage: "FINAL", Count: 2},
		{Key: "sf", Name: "Final four", Type: CallTeamset, Points: 10, Stage: "SF", Count: 4},
		{Key: "third", Name: "Bronze match", Type: CallTeamset, Points: 5, Stage: "3RD", Count: 2},
	}}
	if err := spec.Validate(st); err != nil {
		t.Fatal(err)
	}
	implies, exclusive := spec.Relations(st)
	if got := sortedCopy(implies, "champ"); !eq(got, []string{"final", "sf"}) {
		t.Fatalf("champ implies %v (must not imply the consolation match)", got)
	}
	if got := sortedCopy(implies, "final"); !eq(got, []string{"sf"}) {
		t.Fatalf("final implies %v", got)
	}
	if got := sortedCopy(implies, "third"); !eq(got, []string{"sf"}) {
		t.Fatalf("third implies %v", got)
	}
	if len(exclusive) != 0 {
		t.Fatalf("stage sets are never exclusive: %v", exclusive)
	}
}
