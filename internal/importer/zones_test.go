package importer

import (
	"testing"

	"github.com/floholz/matchowl/internal/football"
)

func TestZonesFromStandings(t *testing.T) {
	rows := []football.StandingRow{}
	labels := map[int]string{1: "Promotion - Champions League (League phase)", 2: "Promotion - Champions League (League phase)", 3: "Promotion - Champions League (League phase)", 4: "Promotion - Champions League (League phase)", 5: "Promotion - Europa League (League phase)", 6: "Play-offs", 18: "Relegation - Championship", 19: "Relegation - Championship", 20: "Relegation - Championship"}
	for r := 1; r <= 20; r++ {
		rows = append(rows, football.StandingRow{Rank: r, TeamID: r, Description: labels[r]})
	}
	z := zonesFromStandings([]football.StandingGroup{{Name: "Premier League", Rows: rows}}, 20)
	want := []struct {
		k    string
		f, t int
	}{{"ucl", 1, 4}, {"uel", 5, 5}, {"po", 6, 6}, {"rel", 18, 20}}
	if len(z) != len(want) {
		t.Fatalf("got %d zones: %+v", len(z), z)
	}
	for i, w := range want {
		if z[i].Key != w.k || z[i].From != w.f || z[i].To != w.t {
			t.Errorf("zone %d = %+v, want %+v", i, z[i], w)
		}
	}
	// A split league: championship / relegation groups.
	rows = rows[:0]
	for r := 1; r <= 12; r++ {
		d := "Bundesliga (Relegation Group)"
		if r <= 6 {
			d = "Promotion - Bundesliga (Championship Group)"
		}
		rows = append(rows, football.StandingRow{Rank: r, TeamID: r, Description: d})
	}
	z = zonesFromStandings([]football.StandingGroup{{Name: "Bundesliga", Rows: rows}}, 12)
	if len(z) != 2 || z[0].Key != "champ" || z[0].To != 6 || z[1].Key != "relgroup" || z[1].From != 7 || z[1].To != 12 {
		t.Errorf("split league zones = %+v", z)
	}
	// Two tables (a cup's groups) → no zones.
	if zonesFromStandings([]football.StandingGroup{{Name: "A"}, {Name: "B"}}, 4) != nil {
		t.Error("groups must not get zones")
	}
}
