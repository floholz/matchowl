// Package importer turns an API-Football league season into a seeded
// Matchowl tournament: it derives a structure proposal from the provider's
// round labels (admin-editable before import), then creates the tournament
// record and its teams / groups / matches in one go.
package importer

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/floholz/matchowl/internal/football"
	"github.com/floholz/matchowl/internal/tournaments"
)

// Proposal is what the preview endpoint returns and what the import endpoint
// accepts back (after the admin's edits). Everything the seeder needs beyond
// the editable fields is re-derived from the cached fixtures at import time.
type Proposal struct {
	LeagueID   int    `json:"leagueId"`
	LeagueName string `json:"leagueName"`
	LeagueType string `json:"leagueType"` // League | Cup
	LeagueLogo string `json:"leagueLogo"`
	Country    string `json:"country"`
	Season     int    `json:"season"`
	TeamKind   string `json:"teamKind"` // national | club (majority of teams)

	// Editable.
	Slug        string                   `json:"slug"`
	Name        string                   `json:"name"`
	ShortName   string                   `json:"shortName"`
	ExtIDPrefix string                   `json:"extIdPrefix"`
	StartsAt    string                   `json:"startsAt"` // RFC3339 UTC
	EndsAt      string                   `json:"endsAt"`
	Structure   tournaments.Structure    `json:"structure"`
	Sync        tournaments.Sync         `json:"sync"`
	Forecast    tournaments.ForecastSpec `json:"forecastSpec"`

	// Read-only preview.
	Shape    string         `json:"shape"` // groups+knockout | league | knockout | groups
	Teams    []TeamPreview  `json:"teams"`
	Groups   []GroupPreview `json:"groups"`
	Rounds   []RoundPreview `json:"rounds"`
	Fixtures int            `json:"fixtures"`
	Warnings []string       `json:"warnings"`
}

// TeamPreview is one team as it will be seeded.
type TeamPreview struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Country  string `json:"country"`
	National bool   `json:"national"`
	Logo     string `json:"logo"`
	Group    string `json:"group,omitempty"`
	ISO2     string `json:"iso2,omitempty"`
}

// GroupPreview lists a group's teams by name.
type GroupPreview struct {
	Letter string   `json:"letter"`
	Teams  []string `json:"teams"`
}

// RoundPreview summarises one provider round label.
type RoundPreview struct {
	Label   string `json:"label"`
	Stage   string `json:"stage"`
	Matches int    `json:"matches"`
	First   string `json:"first"` // RFC3339 of the earliest kickoff
}

var (
	groupRoundRe = regexp.MustCompile(`^(?i)group\s+([A-Z0-9]{1,2})(?:\s*-\s*\d+)?$`)
	tableRoundRe = regexp.MustCompile(`^(.+?)\s*-\s*(\d+)$`)
	nonSlugRe    = regexp.MustCompile(`[^a-z0-9]+`)
	nonCodeRe    = regexp.MustCompile(`[^A-Za-z0-9_-]+`)
)

// roundKind classifies a provider round label: group letter (WC-style),
// table round (league season / new UCL "League Stage"), or knockout.
type roundKind int

const (
	kindGroup roundKind = iota
	kindTable
	kindKnockout
)

func classifyRound(label string) (kind roundKind, groupLetter string) {
	if m := groupRoundRe.FindStringSubmatch(label); m != nil {
		return kindGroup, strings.ToUpper(m[1])
	}
	if m := tableRoundRe.FindStringSubmatch(label); m != nil {
		head := strings.ToLower(m[1])
		if strings.Contains(head, "round of") || strings.Contains(head, "final") ||
			strings.Contains(head, "play") || strings.Contains(head, "knockout") ||
			strings.Contains(head, "qualif") || strings.Contains(head, "preliminary") {
			return kindKnockout, ""
		}
		return kindTable, ""
	}
	return kindKnockout, ""
}

// knockoutStage maps a knockout round label to a stage (code, name,
// consolation). Well-known rounds get the canonical codes the scoring config
// and bracket UI already understand; anything else gets a derived code.
func knockoutStage(label string) tournaments.Stage {
	l := strings.ToLower(label)
	switch {
	case strings.Contains(l, "round of 64"):
		return tournaments.Stage{Code: "R64", Name: "Round of 64", Kind: tournaments.KindKnockout}
	case strings.Contains(l, "round of 32"):
		return tournaments.Stage{Code: "R32", Name: "Round of 32", Kind: tournaments.KindKnockout}
	case strings.Contains(l, "round of 16") || strings.Contains(l, "8th finals"):
		return tournaments.Stage{Code: "R16", Name: "Round of 16", Kind: tournaments.KindKnockout}
	case strings.Contains(l, "knockout") && strings.Contains(l, "play"):
		// UCL/UEL/UECL "Knockout Round Play-offs" between league phase and R16.
		return tournaments.Stage{Code: "KOPO", Name: "Knockout play-offs", Kind: tournaments.KindKnockout}
	case strings.Contains(l, "quarter"):
		return tournaments.Stage{Code: "QF", Name: "Quarter-finals", Kind: tournaments.KindKnockout}
	case strings.Contains(l, "semi"):
		return tournaments.Stage{Code: "SF", Name: "Semi-finals", Kind: tournaments.KindKnockout}
	case strings.Contains(l, "3rd place") || strings.Contains(l, "third place") || strings.Contains(l, "bronze"):
		return tournaments.Stage{Code: "3RD", Name: "Third place", Kind: tournaments.KindKnockout, Consolation: true}
	case l == "final" || strings.HasSuffix(l, " final") && !strings.Contains(l, "play"):
		return tournaments.Stage{Code: "FINAL", Name: "Final", Kind: tournaments.KindKnockout}
	}
	code := strings.ToUpper(nonCodeRe.ReplaceAllString(strings.ReplaceAll(label, " ", "_"), ""))
	if len(code) > 12 {
		code = code[:12]
	}
	if code == "" {
		code = "KO"
	}
	name := label
	if len(name) > 64 {
		name = name[:64]
	}
	return tournaments.Stage{Code: code, Name: name, Kind: tournaments.KindKnockout}
}

// StageFor maps a provider round label to the stage its matches belong to:
// the shared "group" stage for group/table rounds, else the knockout stage.
func StageFor(label string) tournaments.Stage {
	if k, _ := classifyRound(label); k != kindKnockout {
		return tournaments.Stage{Code: "group", Name: "Group stage", Kind: tournaments.KindGroup}
	}
	return knockoutStage(label)
}

// QualifierRounds returns the labels of qualifying rounds: knockout rounds
// whose first kickoff precedes the first group/table kickoff (UCL-style
// preliminary / qualifying rounds and the qualifying play-offs). They belong
// to a phase Matchowl doesn't model — only the main competition is imported.
// Label matching alone can't do this: UCL's qualifying round is literally
// "Play-offs" while its post-league knockout is "Knockout Round Play-offs".
// Nil when there's no group/table stage (pure knockout cups keep every round).
func QualifierRounds(fixtures []football.Fixture) map[string]bool {
	kindOf := map[string]roundKind{}
	firstOf := map[string]time.Time{}
	var firstMain time.Time
	haveMain := false
	for _, f := range fixtures {
		k, ok := kindOf[f.Round]
		if !ok {
			k, _ = classifyRound(f.Round)
			kindOf[f.Round] = k
			firstOf[f.Round] = f.Date
		} else if f.Date.Before(firstOf[f.Round]) {
			firstOf[f.Round] = f.Date
		}
		if k != kindKnockout && (!haveMain || f.Date.Before(firstMain)) {
			firstMain, haveMain = f.Date, true
		}
	}
	if !haveMain {
		return nil
	}
	out := map[string]bool{}
	for lbl, k := range kindOf {
		if k == kindKnockout && firstOf[lbl].Before(firstMain) {
			out[lbl] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Slugify lowercases and dashes a name for slugs / club keys.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "é", "e", "è", "e", "á", "a", "í", "i", "ó", "o", "ú", "u", "ñ", "n", "ç", "c").Replace(s)
	s = nonSlugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// seasonLabel renders "2026" for calendar-year seasons and "2026-27" for
// cross-year ones (derived from the provider's season start/end dates).
func seasonLabel(year int, start, end string) string {
	sy, ey := year, year
	if len(start) >= 4 {
		sy, _ = strconv.Atoi(start[:4])
	}
	if len(end) >= 4 {
		ey, _ = strconv.Atoi(end[:4])
	}
	if sy == 0 {
		sy = year
	}
	if ey > sy {
		return fmt.Sprintf("%d-%02d", sy, ey%100)
	}
	return strconv.Itoa(sy)
}

// prefixFor builds an extIdPrefix like "BUN26" / "FWC26" from the league
// name's initials (or first letters) and the season's two-digit year.
func prefixFor(name string, year int) string {
	words := strings.Fields(nonCodeRe.ReplaceAllString(name, " "))
	var b strings.Builder
	if len(words) >= 2 {
		for _, w := range words {
			b.WriteByte(w[0])
			if b.Len() >= 4 {
				break
			}
		}
	} else if len(words) == 1 {
		w := words[0]
		if len(w) > 3 {
			w = w[:3]
		}
		b.WriteString(w)
	} else {
		b.WriteString("T")
	}
	return strings.ToUpper(b.String()) + fmt.Sprintf("%02d", year%100)
}

// Derive builds the proposal for a league season from its fixtures, team
// list and standings. teams / standings may be empty (quota exhausted or the
// season hasn't started) — then teams come from fixtures without
// codes/countries, and group membership is inferred from who plays whom.
func Derive(league football.League, season football.Season, fixtures []football.Fixture, teams []football.Team, standings []football.StandingGroup) *Proposal {
	excluded := QualifierRounds(fixtures)
	mainTeams := map[int]bool{} // ids seen in non-qualifier fixtures
	if len(excluded) > 0 {
		kept := make([]football.Fixture, 0, len(fixtures))
		for _, f := range fixtures {
			if excluded[f.Round] {
				continue
			}
			kept = append(kept, f)
			mainTeams[f.HomeID] = true
			mainTeams[f.AwayID] = true
		}
		fixtures = kept
	}
	p := &Proposal{
		LeagueID: league.ID, LeagueName: league.Name, LeagueType: league.Type,
		LeagueLogo: league.Logo, Country: league.Country, Season: season.Year,
		Fixtures: len(fixtures),
	}
	if len(excluded) > 0 {
		labels := make([]string, 0, len(excluded))
		for l := range excluded {
			labels = append(labels, l)
		}
		sort.Strings(labels)
		p.Warnings = append(p.Warnings, fmt.Sprintf("skipped %d qualifying round(s) played before the main stage: %s", len(labels), strings.Join(labels, ", ")))
	}
	label := seasonLabel(season.Year, season.Start, season.End)
	p.Slug = Slugify(league.Name) + "-" + label
	p.Name = league.Name + " " + strings.ReplaceAll(label, "-", "/")
	p.ShortName = league.Name
	p.ExtIDPrefix = prefixFor(league.Name, season.Year)
	p.Sync = tournaments.Sync{Provider: tournaments.ProviderAPIFootball, APIFootballLeague: league.ID, Season: season.Year}
	p.Forecast = tournaments.ForecastSpec{Mode: tournaments.ForecastNone}

	// Teams: prefer /teams (codes, national flag); merge anything only seen
	// in fixtures.
	byID := map[int]*TeamPreview{}
	for _, t := range teams {
		if len(excluded) > 0 && !mainTeams[t.ID] {
			continue // eliminated in qualifying — never plays in the main stage
		}
		byID[t.ID] = &TeamPreview{ID: t.ID, Name: t.Name, Code: t.Code, Country: t.Country, National: t.National, Logo: t.Logo}
	}
	for _, f := range fixtures {
		for _, side := range []struct {
			id   int
			name string
			logo string
		}{{f.HomeID, f.HomeName, f.HomeLogo}, {f.AwayID, f.AwayName, f.AwayLogo}} {
			if side.id == 0 || side.name == "" {
				continue
			}
			if _, ok := byID[side.id]; !ok {
				byID[side.id] = &TeamPreview{ID: side.id, Name: side.name, Logo: side.logo}
			}
		}
	}
	nationalCount := 0
	for _, t := range byID {
		if t.National {
			nationalCount++
		}
	}
	national := nationalCount*2 > len(byID) // majority national → national-team event
	p.TeamKind = "club"
	if national {
		p.TeamKind = "national"
	}
	for _, t := range byID {
		if national || t.National {
			t.ISO2 = CountryISO2(t.Country)
			if t.ISO2 == "" {
				t.ISO2 = CountryISO2(t.Name)
			}
		}
	}

	// Rounds → kinds; non-knockout fixtures feed the group inference.
	type roundAgg struct {
		label  string
		kind   roundKind
		letter string
		n      int
		first  time.Time
	}
	rounds := map[string]*roundAgg{}
	var order []string
	groupOfTeam := map[int]string{} // provider team id → group letter
	groupGames := map[int]int{}     // team → group-stage matches
	labelled := false               // round labels carried group letters
	var tableFixtures []football.Fixture
	kickoffs := make([]time.Time, 0, len(fixtures))
	for _, f := range fixtures {
		kickoffs = append(kickoffs, f.Date)
		r, ok := rounds[f.Round]
		if !ok {
			k, letter := classifyRound(f.Round)
			r = &roundAgg{label: f.Round, kind: k, letter: letter, first: f.Date}
			rounds[f.Round] = r
			order = append(order, f.Round)
		}
		r.n++
		if f.Date.Before(r.first) {
			r.first = f.Date
		}
		if r.kind == kindKnockout {
			continue
		}
		for _, id := range []int{f.HomeID, f.AwayID} {
			if id != 0 {
				groupGames[id]++
			}
		}
		if r.kind == kindGroup {
			labelled = true
			for _, id := range []int{f.HomeID, f.AwayID} {
				if id != 0 {
					groupOfTeam[id] = r.letter
				}
			}
		} else {
			tableFixtures = append(tableFixtures, f)
		}
	}
	sort.Slice(order, func(i, j int) bool { return rounds[order[i]].first.Before(rounds[order[j]].first) })

	// Table-style rounds ("Group Stage - 1", "Regular Season - 12"): group
	// membership comes from the standings when available, else from the
	// connected components of who-plays-whom (one component = a league).
	if len(tableFixtures) > 0 && !labelled {
		for id, letter := range groupsFromStandings(standings, tableFixtures) {
			groupOfTeam[id] = letter
		}
		if len(groupOfTeam) == 0 {
			for id, letter := range groupsFromComponents(tableFixtures) {
				groupOfTeam[id] = letter
			}
		}
	} else if len(tableFixtures) > 0 && labelled {
		p.Warnings = append(p.Warnings, "both lettered group rounds and table rounds found — table rounds folded into the group stage; check the structure")
		for _, f := range tableFixtures {
			for _, id := range []int{f.HomeID, f.AwayID} {
				if id != 0 {
					if _, ok := groupOfTeam[id]; !ok {
						groupOfTeam[id] = "A"
					}
				}
			}
		}
	}
	distinct := map[string]bool{}
	for _, l := range groupOfTeam {
		distinct[l] = true
	}
	hasGroups := len(distinct) > 1 // several groups → WC/Euro shape
	hasTable := len(distinct) == 1 // one table → league season

	var stages []tournaments.Stage
	seenStage := map[string]bool{}
	for _, lbl := range order {
		r := rounds[lbl]
		var st tournaments.Stage
		if r.kind == kindKnockout {
			st = knockoutStage(lbl)
		} else {
			st = tournaments.Stage{Code: "group", Name: "Group stage", Kind: tournaments.KindGroup}
			if hasTable {
				st.Name = "League"
			}
		}
		if !seenStage[st.Code] {
			seenStage[st.Code] = true
			stages = append(stages, st)
		}
		p.Rounds = append(p.Rounds, RoundPreview{Label: lbl, Stage: st.Code, Matches: r.n, First: r.first.UTC().Format(time.RFC3339)})
	}
	// Group stage must come first in play order (Validate allows one).
	sort.SliceStable(stages, func(i, j int) bool {
		return stages[i].Kind == tournaments.KindGroup && stages[j].Kind != tournaments.KindGroup
	})
	p.Structure.Stages = stages

	if len(groupOfTeam) > 0 {
		groupSize := 0
		perGroup := map[string][]string{}
		for id, letter := range groupOfTeam {
			if t, ok := byID[id]; ok {
				t.Group = letter
				perGroup[letter] = append(perGroup[letter], t.Name)
			}
		}
		games := 0
		for _, n := range groupGames {
			if n > games {
				games = n
			}
		}
		letters := make([]string, 0, len(perGroup))
		for l, names := range perGroup {
			sort.Strings(names)
			letters = append(letters, l)
			if len(names) > groupSize {
				groupSize = len(names)
			}
		}
		sort.Strings(letters)
		for _, l := range letters {
			p.Groups = append(p.Groups, GroupPreview{Letter: l, Teams: perGroup[l]})
		}
		p.Structure.GroupSize = groupSize
		p.Structure.GamesPerTeam = games
		if hasGroups {
			// WC/Euro-style: top 2 advance by default; the admin tunes it.
			p.Structure.DirectQualifiers = 2
			if len(stages) == 1 {
				p.Structure.DirectQualifiers = 0
			}
		}
		if p.Structure.GamesPerTeam == 0 {
			p.Structure.GamesPerTeam = 1
		}
	}

	switch {
	case hasGroups && len(stages) > 1:
		p.Shape = "groups+knockout"
	case hasGroups:
		p.Shape = "groups"
	case hasTable && len(stages) > 1:
		p.Shape = "league+knockout"
	case hasTable:
		p.Shape = "league"
	default:
		p.Shape = "knockout"
	}
	// The full forecast builder (group tables + bracket) fits WC/Euro
	// shapes; everything else starts without a forecast until the admin
	// defines calls.
	if p.Shape == "groups+knockout" {
		p.Forecast.Mode = tournaments.ForecastFull
	}

	if len(kickoffs) > 0 {
		sort.Slice(kickoffs, func(i, j int) bool { return kickoffs[i].Before(kickoffs[j]) })
		p.StartsAt = kickoffs[0].UTC().Format(time.RFC3339)
		p.EndsAt = kickoffs[len(kickoffs)-1].UTC().Format(time.RFC3339)
	}

	p.Teams = make([]TeamPreview, 0, len(byID))
	for _, t := range byID {
		p.Teams = append(p.Teams, *t)
	}
	sort.Slice(p.Teams, func(i, j int) bool {
		if p.Teams[i].Group != p.Teams[j].Group {
			return p.Teams[i].Group < p.Teams[j].Group
		}
		return p.Teams[i].Name < p.Teams[j].Name
	})

	if len(fixtures) == 0 {
		p.Warnings = append(p.Warnings, "no fixtures published yet for this season — nothing to seed")
	}
	tbd := 0
	for _, f := range fixtures {
		if f.HomeID == 0 || f.AwayID == 0 {
			tbd++
		}
	}
	if tbd > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d fixtures have TBD teams (knockout slots) — they fill in via sync", tbd))
	}
	if len(teams) == 0 && len(byID) > 0 {
		p.Warnings = append(p.Warnings, "team list unavailable — codes/flags derived from names")
	}
	return p
}

// groupLetter turns a standings group name ("Group A", "Group 1", "Serie A")
// into a letter; falls back to the i-th letter when the name has no obvious
// short suffix.
func groupLetter(name string, i int) string {
	n := strings.TrimSpace(name)
	if idx := strings.LastIndex(strings.ToLower(n), "group "); idx >= 0 {
		if suf := strings.TrimSpace(n[idx+6:]); len(suf) <= 2 && suf != "" {
			return strings.ToUpper(suf)
		}
	}
	return string(rune('A' + i%26))
}

// groupsFromStandings maps provider team ids to group letters using the
// standings tables — only for teams that actually appear in table fixtures.
func groupsFromStandings(standings []football.StandingGroup, fixtures []football.Fixture) map[int]string {
	if len(standings) == 0 {
		return nil
	}
	inTable := map[int]bool{}
	for _, f := range fixtures {
		inTable[f.HomeID] = true
		inTable[f.AwayID] = true
	}
	out := map[int]string{}
	for i, g := range standings {
		letter := groupLetter(g.Name, i)
		for _, id := range g.TeamIDs {
			if inTable[id] {
				out[id] = letter
			}
		}
	}
	return out
}

// groupsFromComponents infers groups as connected components of the
// who-plays-whom graph, lettered A.. in order of each component's first
// kickoff (WC/Euro schedules open with Group A).
func groupsFromComponents(fixtures []football.Fixture) map[int]string {
	parent := map[int]int{}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	first := map[int]time.Time{}
	for _, f := range fixtures {
		if f.HomeID == 0 || f.AwayID == 0 {
			continue
		}
		for _, id := range []int{f.HomeID, f.AwayID} {
			if _, ok := parent[id]; !ok {
				parent[id] = id
			}
			if t, ok := first[id]; !ok || f.Date.Before(t) {
				first[id] = f.Date
			}
		}
		parent[find(f.HomeID)] = find(f.AwayID)
	}
	compFirst := map[int]time.Time{}
	for id := range parent {
		r := find(id)
		if t, ok := compFirst[r]; !ok || first[id].Before(t) {
			compFirst[r] = first[id]
		}
	}
	roots := make([]int, 0, len(compFirst))
	for r := range compFirst {
		roots = append(roots, r)
	}
	sort.Slice(roots, func(i, j int) bool {
		if !compFirst[roots[i]].Equal(compFirst[roots[j]]) {
			return compFirst[roots[i]].Before(compFirst[roots[j]])
		}
		return roots[i] < roots[j]
	})
	letterOf := map[int]string{}
	for i, r := range roots {
		letterOf[r] = string(rune('A' + i%26))
	}
	out := map[int]string{}
	for id := range parent {
		out[id] = letterOf[find(id)]
	}
	return out
}
