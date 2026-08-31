package importer

import (
	"strings"
	"testing"
	"time"

	"github.com/floholz/matchowl/internal/football"
	"github.com/floholz/matchowl/internal/tournaments"
)

func fx(id int, day int, round string, home, away int) football.Fixture {
	base := time.Date(2026, 6, 11, 18, 0, 0, 0, time.UTC)
	f := football.Fixture{ID: id, Date: base.AddDate(0, 0, day), Round: round, Status: "NS"}
	if home > 0 {
		f.HomeID, f.HomeName = home, teamName(home)
	}
	if away > 0 {
		f.AwayID, f.AwayName = away, teamName(away)
	}
	return f
}

func teamName(id int) string { return "Team " + string(rune('A'+id-1)) }

func teams(ids ...int) []football.Team {
	out := make([]football.Team, 0, len(ids))
	for _, id := range ids {
		out = append(out, football.Team{ID: id, Name: teamName(id), Code: "T" + string(rune('A'+id-1)), National: true, Country: "Germany"})
	}
	return out
}

func stageCodes(s tournaments.Structure) []string {
	out := make([]string, 0, len(s.Stages))
	for _, st := range s.Stages {
		out = append(out, st.Code)
	}
	return out
}

func eqStrings(a, b []string) bool {
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

func TestDeriveGroupsAndKnockout(t *testing.T) {
	// Two groups of 4 (single round-robin = 3 games each), SF, 3rd, Final.
	var fixtures []football.Fixture
	id := 1
	groups := map[string][]int{"A": {1, 2, 3, 4}, "B": {5, 6, 7, 8}}
	for letter, ts := range groups {
		day := 0
		for i := 0; i < 4; i++ {
			for j := i + 1; j < 4; j++ {
				fixtures = append(fixtures, fx(id, day, "Group "+letter+" - 1", ts[i], ts[j]))
				id++
				day++
			}
		}
	}
	fixtures = append(fixtures,
		fx(id, 20, "Semi-finals", 0, 0), fx(id+1, 20, "Semi-finals", 0, 0),
		fx(id+2, 24, "3rd Place Final", 0, 0), fx(id+3, 25, "Final", 0, 0))

	lg := football.League{ID: 4, Name: "Euro Championship", Type: "Cup", Country: "World"}
	p := Derive(lg, football.Season{Year: 2028, Start: "2028-06-10", End: "2028-07-10"}, fixtures, teams(1, 2, 3, 4, 5, 6, 7, 8), nil)

	if p.Shape != "groups+knockout" {
		t.Fatalf("shape = %q", p.Shape)
	}
	if got := stageCodes(p.Structure); !eqStrings(got, []string{"group", "SF", "3RD", "FINAL"}) {
		t.Fatalf("stages = %v", got)
	}
	if p.Structure.GroupSize != 4 || p.Structure.GamesPerTeam != 3 || p.Structure.DirectQualifiers != 2 {
		t.Fatalf("group shape = %+v", p.Structure)
	}
	if len(p.Groups) != 2 || p.Groups[0].Letter != "A" || len(p.Groups[0].Teams) != 4 {
		t.Fatalf("groups = %+v", p.Groups)
	}
	if p.Structure.Stages[2].Code != "3RD" || !p.Structure.Stages[2].Consolation {
		t.Fatalf("3rd place should be consolation: %+v", p.Structure.Stages[2])
	}
	if p.Slug != "euro-championship-2028" || p.ExtIDPrefix != "EC28" {
		t.Fatalf("slug/prefix = %q %q", p.Slug, p.ExtIDPrefix)
	}
	if p.Forecast.Mode != tournaments.ForecastFull {
		t.Fatalf("forecast mode = %q", p.Forecast.Mode)
	}
	if p.Teams[0].ISO2 != "de" {
		t.Fatalf("national team iso2 = %q", p.Teams[0].ISO2)
	}
	if err := p.Structure.Validate(); err != nil {
		t.Fatalf("structure invalid: %v", err)
	}
	if len(p.Warnings) == 0 {
		t.Fatalf("expected a TBD warning")
	}
}

func TestDeriveLeagueSeason(t *testing.T) {
	// 4 clubs, double round-robin = 6 games each, "Regular Season - n".
	var fixtures []football.Fixture
	id, day := 1, 0
	for r := 1; r <= 6; r++ {
		fixtures = append(fixtures, fx(id, day, "Regular Season - "+string(rune('0'+r)), 1+(r%4), 1+((r+1)%4)))
		id++
		fixtures = append(fixtures, fx(id, day, "Regular Season - "+string(rune('0'+r)), 1+((r+2)%4), 1+((r+3)%4)))
		id++
		day += 7
	}
	lg := football.League{ID: 78, Name: "Bundesliga", Type: "League", Country: "Germany"}
	clubs := teams(1, 2, 3, 4)
	for i := range clubs {
		clubs[i].National = false
	}
	p := Derive(lg, football.Season{Year: 2026, Start: "2026-08-14", End: "2027-05-22"}, fixtures, clubs, nil)

	if p.Shape != "league" {
		t.Fatalf("shape = %q", p.Shape)
	}
	if got := stageCodes(p.Structure); !eqStrings(got, []string{"group"}) {
		t.Fatalf("stages = %v", got)
	}
	if p.Structure.GroupSize != 4 || p.Structure.GamesPerTeam != 6 || p.Structure.DirectQualifiers != 0 {
		t.Fatalf("league shape = %+v", p.Structure)
	}
	if len(p.Groups) != 1 || p.Groups[0].Letter != "A" {
		t.Fatalf("groups = %+v", p.Groups)
	}
	if p.Slug != "bundesliga-2026-27" || p.Name != "Bundesliga 2026/27" || p.ExtIDPrefix != "BUN26" {
		t.Fatalf("identity = %q %q %q", p.Slug, p.Name, p.ExtIDPrefix)
	}
	if p.Forecast.Mode != tournaments.ForecastNone {
		t.Fatalf("forecast mode = %q", p.Forecast.Mode)
	}
	if p.Teams[0].ISO2 != "" {
		t.Fatalf("clubs should not get iso2, got %q", p.Teams[0].ISO2)
	}
	if err := p.Structure.Validate(); err != nil {
		t.Fatalf("structure invalid: %v", err)
	}
}

func TestDeriveKnockoutCup(t *testing.T) {
	fixtures := []football.Fixture{
		fx(1, 0, "1st Round", 1, 2), fx(2, 0, "1st Round", 3, 4),
		fx(3, 30, "Round of 16", 0, 0), fx(4, 60, "Quarter-finals", 0, 0),
		fx(5, 90, "Semi-finals", 0, 0), fx(6, 120, "Final", 0, 0),
	}
	p := Derive(football.League{ID: 81, Name: "DFB Pokal", Type: "Cup"}, football.Season{Year: 2026}, fixtures, nil, nil)
	if p.Shape != "knockout" {
		t.Fatalf("shape = %q", p.Shape)
	}
	if got := stageCodes(p.Structure); !eqStrings(got, []string{"1ST_ROUND", "R16", "QF", "SF", "FINAL"}) {
		t.Fatalf("stages = %v", got)
	}
	if p.Structure.GroupSize != 0 || len(p.Groups) != 0 {
		t.Fatalf("no groups expected: %+v", p.Structure)
	}
	if err := p.Structure.Validate(); err != nil {
		t.Fatalf("structure invalid: %v", err)
	}
	// Teams only from fixtures (no /teams) → still listed, no codes.
	if len(p.Teams) != 4 || p.Teams[0].Code != "" {
		t.Fatalf("teams = %+v", p.Teams)
	}
}

// WC-2022 shape as API-Football labels it: "Group Stage - n" without letters.
func groupStageFixtures() []football.Fixture {
	var fixtures []football.Fixture
	id := 1
	groups := [][]int{{1, 2, 3, 4}, {5, 6, 7, 8}}
	for gi, ts := range groups {
		day := gi // group A opens the tournament
		for i := 0; i < 4; i++ {
			for j := i + 1; j < 4; j++ {
				fixtures = append(fixtures, fx(id, day, "Group Stage - 1", ts[i], ts[j]))
				id++
				day += 2
			}
		}
	}
	return append(fixtures, fx(id, 30, "Semi-finals", 0, 0), fx(id+1, 35, "Final", 0, 0))
}

func TestDeriveUnletteredGroupsFromStandings(t *testing.T) {
	standings := []football.StandingGroup{
		{Name: "Group B", TeamIDs: []int{5, 6, 7, 8}},
		{Name: "Group A", TeamIDs: []int{1, 2, 3, 4}},
	}
	p := Derive(football.League{ID: 1, Name: "World Cup"}, football.Season{Year: 2022}, groupStageFixtures(), teams(1, 2, 3, 4, 5, 6, 7, 8), standings)
	if p.Shape != "groups+knockout" {
		t.Fatalf("shape = %q", p.Shape)
	}
	if len(p.Groups) != 2 || p.Groups[0].Letter != "A" || p.Groups[0].Teams[0] != "Team A" || p.Groups[1].Letter != "B" {
		t.Fatalf("groups = %+v", p.Groups)
	}
	if p.Structure.GroupSize != 4 || p.Structure.GamesPerTeam != 3 {
		t.Fatalf("shape = %+v", p.Structure)
	}
}

func TestDeriveUnletteredGroupsFromSchedule(t *testing.T) {
	// No standings (season not started): infer from who plays whom; the
	// component that kicks off first is Group A.
	p := Derive(football.League{ID: 1, Name: "World Cup"}, football.Season{Year: 2026}, groupStageFixtures(), teams(1, 2, 3, 4, 5, 6, 7, 8), nil)
	if p.Shape != "groups+knockout" {
		t.Fatalf("shape = %q", p.Shape)
	}
	if len(p.Groups) != 2 || p.Groups[0].Letter != "A" || p.Groups[0].Teams[0] != "Team A" || p.Groups[1].Teams[0] != "Team E" {
		t.Fatalf("groups = %+v", p.Groups)
	}
}

// UCL-style season: qualifying rounds (including the ambiguous "Play-offs")
// before a single league phase, knockout rounds after it. Qualifiers are
// excluded; qualifier-only teams are not seeded.
func TestDeriveUCLQualifiersExcluded(t *testing.T) {
	fixtures := []football.Fixture{
		fx(1, -40, "Preliminary Round", 7, 8),
		fx(2, -30, "1st Qualifying Round", 5, 7),
		fx(3, -10, "Play-offs", 5, 6), // qualifying play-off, both teams reach the league phase
		fx(4, 0, "League Stage - 1", 1, 2), fx(5, 0, "League Stage - 1", 3, 4), fx(6, 0, "League Stage - 1", 5, 6),
		fx(7, 7, "League Stage - 2", 2, 3), fx(8, 7, "League Stage - 2", 4, 5), fx(9, 7, "League Stage - 2", 6, 1),
		fx(10, 14, "League Stage - 3", 1, 3), fx(11, 14, "League Stage - 3", 2, 5), fx(12, 14, "League Stage - 3", 4, 6),
		fx(13, 100, "Knockout Round Play-offs", 0, 0),
		fx(14, 110, "Semi-finals", 0, 0),
		fx(15, 120, "Final", 0, 0),
	}
	standings := []football.StandingGroup{{Name: "League Stage", TeamIDs: []int{1, 2, 3, 4, 5, 6}}}
	p := Derive(football.League{ID: 2, Name: "UEFA Champions League", Type: "Cup"}, football.Season{Year: 2026, Start: "2026-09-01", End: "2027-05-30"}, fixtures, teams(1, 2, 3, 4, 5, 6, 7, 8), standings)

	if p.Shape != "league+knockout" {
		t.Fatalf("shape = %q", p.Shape)
	}
	if got := stageCodes(p.Structure); !eqStrings(got, []string{"group", "KOPO", "SF", "FINAL"}) {
		t.Fatalf("stages = %v", got)
	}
	if len(p.Teams) != 6 {
		t.Fatalf("qualifier-only teams should be dropped, got %d teams: %+v", len(p.Teams), p.Teams)
	}
	if p.Structure.GroupSize != 6 || p.Structure.GamesPerTeam != 3 {
		t.Fatalf("league phase shape = %+v", p.Structure)
	}
	if p.Fixtures != 12 {
		t.Fatalf("fixtures = %d, want 12 (qualifiers excluded)", p.Fixtures)
	}
	found := false
	for _, w := range p.Warnings {
		if strings.Contains(w, "qualifying") && strings.Contains(w, "Play-offs") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a qualifying-rounds warning, got %v", p.Warnings)
	}
	if err := p.Structure.Validate(); err != nil {
		t.Fatalf("structure invalid: %v", err)
	}
}

func TestCodeFor(t *testing.T) {
	used := map[string]bool{}
	if c := codeFor("Bayern München", "BAY", used); c != "BAY" {
		t.Fatalf("got %q", c)
	}
	if c := codeFor("Bayer Leverkusen", "BAY", used); c != "BA0" {
		t.Fatalf("dedupe got %q", c)
	}
	if c := codeFor("VfB Stuttgart", "", used); c != "VFB" {
		t.Fatalf("derived got %q", c)
	}
}
