package teams

import "testing"

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"FK Austria Wien":     "fk-austria-wien",
		"Bayern München":      "bayern-muenchen",
		"  Atlético Madrid":   "atletico-madrid",
		"Paris Saint-Germain": "paris-saint-germain",
		"???":                 "",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTable(t *testing.T) {
	tbl := Table([]Result{
		{Home: "a", Away: "b", HomeGoals: 2, AwayGoals: 0, Played: true},
		{Home: "c", Away: "a", HomeGoals: 1, AwayGoals: 1, Played: true},
		{Home: "b", Away: "c", HomeGoals: 3, AwayGoals: 1, Played: true},
		{Home: "d", Away: "a", Played: false},
	})
	if len(tbl) != 4 {
		t.Fatalf("rows = %d, want 4 (unplayed teams get a row)", len(tbl))
	}
	if tbl[0].Team != "a" || tbl[0].Pts != 4 || tbl[0].Played != 2 {
		t.Errorf("first = %+v, want a with 4 pts from 2", tbl[0])
	}
	if tbl[1].Team != "b" || tbl[2].Team != "c" || tbl[3].Team != "d" {
		t.Errorf("order = %s %s %s, want b c d", tbl[1].Team, tbl[2].Team, tbl[3].Team)
	}
}
