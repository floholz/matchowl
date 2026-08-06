// Package seed populates a tournament's teams, groups and fixture list from
// openfootball-format JSON (fixtures doc + team meta). The embedded WC2026
// dataset seeds the first tournament on first boot (idempotent: skipped once
// that tournament has teams); new tournaments are seeded through the admin
// endpoint with uploaded data.
package seed

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/tournaments"
	"github.com/floholz/matchowl/internal/users"
)

//go:embed data/worldcup2026.json data/teams_meta2026.json
var dataFS embed.FS

type ofMatch struct {
	Round  string `json:"round"`
	Num    int    `json:"num"`
	Date   string `json:"date"`
	Time   string `json:"time"`
	Team1  string `json:"team1"`
	Team2  string `json:"team2"`
	Group  string `json:"group"`
	Ground string `json:"ground"`
}

type ofTeam struct {
	Name        string `json:"name"`
	FifaCode    string `json:"fifa_code"`
	FlagUnicode string `json:"flag_unicode"`
	Group       string `json:"group"`
	Confed      string `json:"confed"`
}

var (
	flagCPRe   = regexp.MustCompile(`1F1[0-9A-Fa-f]{2}`)
	roundStage = map[string]string{
		"Round of 32":           "R32",
		"Round of 16":           "R16",
		"Quarter-final":         "QF",
		"Semi-final":            "SF",
		"Match for third place": "3RD",
		"Final":                 "FINAL",
	}
)

// HomeNationISO maps FIFA codes that have no ISO-3166 country (UK home
// nations use emoji tag-sequences, not regional indicators) to the
// flag-icons file name.
var HomeNationISO = map[string]string{
	"ENG": "gb-eng",
	"SCO": "gb-sct",
	"WAL": "gb-wls",
	"NIR": "gb-nir",
}

// iso2FromFlag turns openfootball's "\u{1F1F2}\u{1F1FD}" regional-indicator
// escape into the ISO-3166 alpha-2 code ("mx") used for the bundled flag SVGs.
func iso2FromFlag(flagUnicode string) string {
	cps := flagCPRe.FindAllString(flagUnicode, 2)
	if len(cps) != 2 {
		return ""
	}
	var sb strings.Builder
	for _, c := range cps {
		v, err := strconv.ParseInt(c, 16, 32)
		if err != nil {
			return ""
		}
		sb.WriteRune(rune('a' + (v - 0x1F1E6)))
	}
	return sb.String()
}

// parseKickoff combines "2026-06-11" + "13:00 UTC-6" into a UTC time.
func parseKickoff(date, tm string) (time.Time, error) {
	parts := strings.Fields(tm) // ["13:00", "UTC-6"]
	clock := "00:00"
	offset := 0
	if len(parts) >= 1 {
		clock = parts[0]
	}
	if len(parts) >= 2 {
		off := strings.TrimPrefix(parts[1], "UTC")
		if n, err := strconv.Atoi(off); err == nil {
			offset = n
		}
	}
	loc := time.FixedZone("seed", offset*3600)
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// RoundStage maps an openfootball round label to our stage code.
func RoundStage(round string) string {
	if s, ok := roundStage[round]; ok {
		return s
	}
	return "group"
}

// ExtID is the deterministic match id shared by the seed and the live-results
// sync, so openfootball live matches map 1:1 onto our rows (no name aliases).
// The prefix comes from the tournament record's extIdPrefix.
func ExtID(prefix, round string, num int, group, team1, team2 string) string {
	stage := RoundStage(round)
	if stage == "group" {
		return fmt.Sprintf("%s-G-%s-%s-%s",
			prefix, strings.ReplaceAll(group, " ", ""), slug(team1), slug(team2))
	}
	if num > 0 {
		return fmt.Sprintf("%s-K-%d", prefix, num)
	}
	return prefix + "-K-" + stage
}

// displayNames overrides the stored display name for teams whose preferred label
// differs from the openfootball seed name. The seed/openfootball name is kept for
// all *matching* (byName fixture resolution + the ExtID slug that openfootball
// live results map onto); only the user-facing `name` is swapped. Keep this in
// sync with the rename migration that fixes already-seeded databases.
var displayNames = map[string]string{
	"Turkey": "Türkiye",
}

// displayName returns the preferred user-facing label for an openfootball team
// name (the name itself when there's no override).
func displayName(seedName string) string {
	if d, ok := displayNames[seedName]; ok {
		return d
	}
	return seedName
}

func slug(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, s)
}

// DefaultScoringConfig — the agreed rules; tunable without code changes
// (per-League overrides reference a different scoring_configs record).
// Max 6 per game (group 1/X/2, KO = who advances; no separate advancer / ET
// bonus). Forecast: exact group position (+ perfect bonus), +advance per
// correctly-predicted advancer, escalating KO rounds.
const DefaultScoringConfig = `{
  "match": {
    "tendency": 3,
    "exact": 1,
    "totalGoals": 1,
    "goalDiff": 1
  },
  "forecast": {
    "groupPosition": 1,
    "perfectGroupBonus": 2,
    "advance": 1,
    "round": { "R32": 1, "R16": 2, "QF": 3, "SF": 5, "FINAL": 8, "CHAMPION": 13 }
  },
  "tiebreakers": ["points", "exactScores", "correctWinners", "goalDiffDeviation", "fewestTips", "earliestEdit"]
}`

// ensureDefaultScoringConfig creates the default scoring config once.
func ensureDefaultScoringConfig(app core.App) error {
	if n, _ := app.CountRecords("scoring_configs"); n > 0 {
		return nil
	}
	col, err := app.FindCollectionByNameOrId("scoring_configs")
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("name", "Default")
	rec.Set("isDefault", true)
	rec.Set("config", DefaultScoringConfig)
	return app.Save(rec)
}

// Run seeds the database if it hasn't been seeded yet.
func Run(app core.App) error {
	if err := ensureDefaultScoringConfig(app); err != nil {
		return err
	}

	// The wc2026 tournament record is created by migration 0029; every seeded
	// row is scoped to it. Link its scoring config now that the default
	// config exists (fresh DBs run the migration before the seeder).
	tournament, err := app.FindFirstRecordByFilter("tournaments",
		"slug = {:s}", map[string]any{"s": "wc2026"})
	if err != nil {
		return fmt.Errorf("seed: wc2026 tournament record missing: %w", err)
	}
	if tournament.GetString("scoringConfig") == "" {
		if def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true"); err == nil {
			tournament.Set("scoringConfig", def.Id)
			if err := app.Save(tournament); err != nil {
				return err
			}
		}
	}

	if n, _ := app.CountRecords("teams", dbx.HashExp{"tournament": tournament.Id}); n > 0 {
		return nil // already seeded
	}

	teamsRaw, err := dataFS.ReadFile("data/teams_meta2026.json")
	if err != nil {
		return err
	}
	matchesRaw, err := dataFS.ReadFile("data/worldcup2026.json")
	if err != nil {
		return err
	}
	return SeedTournament(app, tournament, teamsRaw, matchesRaw)
}

// SeedTournament seeds one tournament's teams, groups and fixtures from
// openfootball-format JSON: teamsRaw is the team meta array (name, fifa_code,
// flag_unicode, group, confed), fixturesRaw the fixtures doc ({"matches":
// [...]}) whose round labels map to stage codes via RoundStage. Refuses when
// the tournament already has teams, and rejects fixtures whose derived stage
// is missing from the tournament's structure.
func SeedTournament(app core.App, tournament *core.Record, teamsRaw, fixturesRaw []byte) error {
	if n, _ := app.CountRecords("teams", dbx.HashExp{"tournament": tournament.Id}); n > 0 {
		return fmt.Errorf("tournament %s already has teams", tournament.GetString("slug"))
	}
	st, err := tournaments.StructureOf(tournament)
	if err != nil {
		return fmt.Errorf("structure: %w", err)
	}
	validStage := map[string]bool{}
	for _, sc := range st.StageCodes() {
		validStage[sc] = true
	}

	var ofTeams []ofTeam
	if err := json.Unmarshal(teamsRaw, &ofTeams); err != nil {
		return fmt.Errorf("teams meta: %w", err)
	}
	if len(ofTeams) == 0 {
		return fmt.Errorf("teams meta: empty")
	}
	var wc struct {
		Matches []ofMatch `json:"matches"`
	}
	if err := json.Unmarshal(fixturesRaw, &wc); err != nil {
		return fmt.Errorf("fixtures: %w", err)
	}
	if len(wc.Matches) == 0 {
		return fmt.Errorf("fixtures: empty")
	}
	for _, m := range wc.Matches {
		if stage := RoundStage(m.Round); !validStage[stage] {
			return fmt.Errorf("fixtures: round %q maps to stage %q which is not in the tournament structure", m.Round, stage)
		}
	}

	teamsCol, err := app.FindCollectionByNameOrId("teams")
	if err != nil {
		return err
	}

	return app.RunInTransaction(func(txApp core.App) error {
		// Teams, keyed by openfootball display name for fixture resolution.
		byName := map[string]*core.Record{}
		groupTeams := map[string][]string{}
		for _, t := range ofTeams {
			rec := core.NewRecord(teamsCol)
			iso2 := iso2FromFlag(t.FlagUnicode)
			if h, ok := HomeNationISO[t.FifaCode]; ok {
				iso2 = h
			}
			rec.Set("tournament", tournament.Id)
			rec.Set("fifaCode", t.FifaCode)
			// Display name may be overridden (e.g. Türkiye); byName + ExtID below
			// stay on the openfootball name so live-results matching is unchanged.
			rec.Set("name", displayName(t.Name))
			rec.Set("iso2", iso2)
			rec.Set("confederation", t.Confed)
			if err := txApp.Save(rec); err != nil {
				return fmt.Errorf("save team %s: %w", t.Name, err)
			}
			byName[t.Name] = rec
			groupTeams[t.Group] = append(groupTeams[t.Group], rec.Id)
		}

		// Tournament groups A..L.
		groupsCol, err := txApp.FindCollectionByNameOrId("tournament_groups")
		if err != nil {
			return err
		}
		for letter, ids := range groupTeams {
			rec := core.NewRecord(groupsCol)
			rec.Set("tournament", tournament.Id)
			rec.Set("letter", letter)
			rec.Set("teams", ids)
			if err := txApp.Save(rec); err != nil {
				return fmt.Errorf("save group %s: %w", letter, err)
			}
		}

		// Matches.
		matchesCol, err := txApp.FindCollectionByNameOrId("matches")
		if err != nil {
			return err
		}
		for _, m := range wc.Matches {
			stage := "group"
			if s, ok := roundStage[m.Round]; ok {
				stage = s
			}
			kickoff, err := parseKickoff(m.Date, m.Time)
			if err != nil {
				return fmt.Errorf("parse kickoff %q %q: %w", m.Date, m.Time, err)
			}
			rec := core.NewRecord(matchesCol)
			rec.Set("tournament", tournament.Id)
			rec.Set("extId", ExtID(tournament.GetString("extIdPrefix"), m.Round, m.Num, m.Group, m.Team1, m.Team2))
			rec.Set("stage", stage)
			rec.Set("num", m.Num)
			rec.Set("roundLabel", m.Round)
			rec.Set("kickoff", kickoff)
			rec.Set("status", "scheduled")
			if stage == "group" {
				rec.Set("groupLetter", strings.TrimPrefix(m.Group, "Group "))
				if h, ok := byName[m.Team1]; ok {
					rec.Set("homeTeam", h.Id)
				}
				if a, ok := byName[m.Team2]; ok {
					rec.Set("awayTeam", a.Id)
				}
			} else {
				// Knockout: teams unknown until results resolve; keep the
				// openfootball placeholder labels ("1A", "3A/B/C/D/F", "W74").
				rec.Set("homeLabel", m.Team1)
				rec.Set("awayLabel", m.Team2)
			}
			if err := txApp.Save(rec); err != nil {
				return fmt.Errorf("save match %s: %w", rec.GetString("extId"), err)
			}
		}
		return nil
	})
}

// Register wires the admin seeding endpoint for new tournaments.
// POST /api/admin/tournaments/{id}/seed with {"teams": [...], "fixtures":
// {"matches": [...]}} in openfootball format.
func Register(app core.App, se *core.ServeEvent) {
	se.Router.POST("/api/admin/tournaments/{id}/seed", func(e *core.RequestEvent) error {
		if e.Auth == nil || !users.IsAdmin(e.Auth) {
			return apis.NewForbiddenError("admin only", nil)
		}
		t, err := app.FindRecordById("tournaments", e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		raw, err := io.ReadAll(io.LimitReader(e.Request.Body, 4<<20))
		if err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		var body struct {
			Teams    json.RawMessage `json:"teams"`
			Fixtures json.RawMessage `json:"fixtures"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if len(body.Teams) == 0 || len(body.Fixtures) == 0 {
			return apis.NewBadRequestError("teams and fixtures are required", nil)
		}
		if err := SeedTournament(app, t, body.Teams, body.Fixtures); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		nTeams, _ := app.CountRecords("teams", dbx.HashExp{"tournament": t.Id})
		nMatches, _ := app.CountRecords("matches", dbx.HashExp{"tournament": t.Id})
		return e.JSON(http.StatusOK, map[string]any{
			"status": "ok", "teams": nTeams, "matches": nMatches,
		})
	}).Bind(apis.RequireAuth())
}
