package sync

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/tournaments"
)

// euroStructure: 4-team groups, 3 games each, top 2 advance + best thirds
// (count 2 for a compact test), 3-1-0 points, no official table.
func euroStructure() *tournaments.Structure {
	s := &tournaments.Structure{
		Stages: []tournaments.Stage{
			{Code: "group", Name: "Group stage", Kind: tournaments.KindGroup},
			{Code: "SF", Name: "Semi-finals", Kind: tournaments.KindKnockout},
			{Code: "FINAL", Name: "Final", Kind: tournaments.KindKnockout},
		},
		GroupSize: 4, GamesPerTeam: 3, DirectQualifiers: 2,
		ExtraQualifiers: &tournaments.ExtraQualifiers{FromPosition: 3, Count: 2},
	}
	s.Normalize()
	return s
}

func matchesCol() *core.Collection {
	col := core.NewBaseCollection("matches")
	col.Fields.Add(&core.TextField{Name: "stage"})
	col.Fields.Add(&core.TextField{Name: "groupLetter"})
	col.Fields.Add(&core.TextField{Name: "homeTeam"})
	col.Fields.Add(&core.TextField{Name: "awayTeam"})
	col.Fields.Add(&core.NumberField{Name: "ftHome", OnlyInt: true})
	col.Fields.Add(&core.NumberField{Name: "ftAway", OnlyInt: true})
	col.Fields.Add(&core.TextField{Name: "finalizedAt"})
	col.Fields.Add(&core.NumberField{Name: "num", OnlyInt: true})
	return col
}

// groupFixtures returns the 6 finished round-robin matches of a 4-team group
// where team i beats team j whenever i < j, with score margins that make
// every position unambiguous: teams[0] 9pts, teams[1] 6, teams[2] 3, teams[3] 0.
// margin skews GD so tests can order thirds across groups.
func groupFixtures(col *core.Collection, letter string, teams [4]string, margin int) []*core.Record {
	var out []*core.Record
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			m := core.NewRecord(col)
			m.Set("stage", "group")
			m.Set("groupLetter", letter)
			m.Set("homeTeam", teams[i])
			m.Set("awayTeam", teams[j])
			m.Set("ftHome", margin)
			m.Set("ftAway", 0)
			m.Set("finalizedAt", "2028-06-20 20:00:00.000Z")
			out = append(out, m)
		}
	}
	return out
}

func TestGroupStandingsEuroShape(t *testing.T) {
	st := euroStructure()
	col := matchesCol()

	// Every third has 3 pts (beat only the 4th); their GD is -margin, so the
	// SMALLEST margin ranks best: C3 (-1) > B3 (-2) > A3 (-3). With Count=2
	// only C3 and B3 qualify.
	var ms []*core.Record
	ms = append(ms, groupFixtures(col, "A", [4]string{"A1", "A2", "A3", "A4"}, 3)...)
	ms = append(ms, groupFixtures(col, "B", [4]string{"B1", "B2", "B3", "B4"}, 2)...)
	ms = append(ms, groupFixtures(col, "C", [4]string{"C1", "C2", "C3", "C4"}, 1)...)

	positions, extras, extraTeam := groupStandings(ms, st)

	for _, want := range []struct {
		pos    int
		letter string
		team   string
	}{
		{1, "A", "A1"}, {2, "A", "A2"}, {3, "A", "A3"}, {4, "A", "A4"},
		{1, "B", "B1"}, {2, "C", "C2"},
	} {
		if got := positions[want.pos][want.letter]; got != want.team {
			t.Errorf("positions[%d][%s] = %q, want %q", want.pos, want.letter, got, want.team)
		}
	}

	if len(extras) != 2 {
		t.Fatalf("extras = %d, want 2 (count cap)", len(extras))
	}
	if extras[0].team != "C3" || extras[1].team != "B3" {
		t.Errorf("extras ranked %q,%q, want C3,B3", extras[0].team, extras[1].team)
	}
	for _, l := range []string{"A", "B", "C"} {
		if extraTeam[l] != l+"3" {
			t.Errorf("extraTeam[%s] = %q, want %s3", l, extraTeam[l], l)
		}
	}
}

func TestGroupStandingsIncompleteGroup(t *testing.T) {
	st := euroStructure()
	col := matchesCol()

	// Only 5 of group A's 6 matches finished → no positions for A.
	ms := groupFixtures(col, "A", [4]string{"A1", "A2", "A3", "A4"}, 1)[:5]
	positions, extras, _ := groupStandings(ms, st)
	if len(positions) != 0 {
		t.Errorf("incomplete group produced positions: %v", positions)
	}
	if len(extras) != 0 {
		t.Errorf("incomplete group produced extras: %v", extras)
	}
}

func TestGroupStandingsNoGroups(t *testing.T) {
	st := &tournaments.Structure{
		Stages: []tournaments.Stage{
			{Code: "SF", Name: "Semi-finals", Kind: tournaments.KindKnockout},
			{Code: "FINAL", Name: "Final", Kind: tournaments.KindKnockout},
		},
	}
	st.Normalize()
	positions, extras, extraTeam := groupStandings(nil, st)
	if len(positions) != 0 || len(extras) != 0 || len(extraTeam) != 0 {
		t.Error("group-less structure must yield empty standings")
	}
}
