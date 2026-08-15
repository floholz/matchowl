package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/football"
	"github.com/floholz/matchowl/internal/tournaments"
	"github.com/floholz/matchowl/internal/users"
)

// cacheTTL bounds how long a fetched season (fixtures + teams) is reused
// between preview and import — every fetch costs API-Football requests.
const cacheTTL = 15 * time.Minute

type seasonData struct {
	league    football.League
	season    football.Season
	fixtures  []football.Fixture
	teams     []football.Team
	standings []football.StandingGroup
	at        time.Time
}

var (
	cacheMu sync.Mutex
	cache   = map[string]*seasonData{}
)

func cacheKey(league, season int) string { return strconv.Itoa(league) + ":" + strconv.Itoa(season) }

// fetchSeason loads (or reuses) one league season's catalog entry, fixtures
// and team list. The team list is best-effort: a quota-exhausted /teams call
// degrades to fixture-derived teams instead of failing the whole preview.
func fetchSeason(ctx context.Context, key string, leagueID, season int) (*seasonData, error) {
	k := cacheKey(leagueID, season)
	cacheMu.Lock()
	if d, ok := cache[k]; ok && time.Since(d.at) < cacheTTL {
		cacheMu.Unlock()
		return d, nil
	}
	cacheMu.Unlock()

	c := football.New(key, leagueID, season)
	leagues, err := c.Leagues(ctx, "", leagueID)
	if err != nil {
		return nil, fmt.Errorf("league %d: %w", leagueID, err)
	}
	if len(leagues) == 0 {
		return nil, fmt.Errorf("league %d not found", leagueID)
	}
	lg := leagues[0]
	var ss football.Season
	found := false
	for _, s := range lg.Seasons {
		if s.Year == season {
			ss, found = s, true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("season %d not offered for %s", season, lg.Name)
	}
	fixtures, err := c.Fixtures(ctx)
	if err != nil {
		return nil, fmt.Errorf("fixtures: %w", err)
	}
	teams, err := c.Teams(ctx, season)
	if err != nil {
		teams = nil // degrade — Derive falls back to fixture-derived teams
	}
	standings, err := c.Standings(ctx, season)
	if err != nil {
		standings = nil // degrade — Derive infers groups from the schedule
	}
	d := &seasonData{league: lg, season: ss, fixtures: fixtures, teams: teams, standings: standings, at: time.Now()}
	cacheMu.Lock()
	cache[k] = d
	cacheMu.Unlock()
	return d, nil
}

// Register wires the admin catalog + import endpoints.
func Register(app core.App, se *core.ServeEvent) {
	adminOnly := func(e *core.RequestEvent) error {
		if e.Auth == nil || !users.IsAdmin(e.Auth) {
			return apis.NewForbiddenError("admin only", nil)
		}
		return e.Next()
	}
	needKey := func(e *core.RequestEvent) (string, error) {
		key := strings.TrimSpace(os.Getenv("API_FOOTBALL_KEY"))
		if key == "" {
			return "", e.JSON(http.StatusBadRequest, map[string]string{
				"error": "API_FOOTBALL_KEY is not set — the catalog needs an API-Football key",
			})
		}
		return key, nil
	}

	fg := se.Router.Group("/api/admin/football")
	fg.Bind(apis.RequireAuth())
	fg.BindFunc(adminOnly)

	// GET /api/admin/football/leagues?search=bundes | ?id=78
	fg.GET("/leagues", func(e *core.RequestEvent) error {
		key, err := needKey(e)
		if err != nil {
			return err
		}
		q := e.Request.URL.Query()
		search := strings.TrimSpace(q.Get("search"))
		id, _ := strconv.Atoi(q.Get("id"))
		if id == 0 && len(search) < 3 {
			return apis.NewBadRequestError("search needs at least 3 characters", nil)
		}
		ctx, cancel := context.WithTimeout(e.Request.Context(), 20*time.Second)
		defer cancel()
		leagues, err := football.New(key, 0, 0).Leagues(ctx, search, id)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		// Cups and leagues that actually have seasons first; alphabetical.
		sort.SliceStable(leagues, func(i, j int) bool { return leagues[i].Name < leagues[j].Name })
		return e.JSON(http.StatusOK, map[string]any{"leagues": leagues})
	})

	// GET /api/admin/football/preview?league=78&season=2023
	fg.GET("/preview", func(e *core.RequestEvent) error {
		key, err := needKey(e)
		if err != nil {
			return err
		}
		q := e.Request.URL.Query()
		leagueID, _ := strconv.Atoi(q.Get("league"))
		season, _ := strconv.Atoi(q.Get("season"))
		if leagueID == 0 || season == 0 {
			return apis.NewBadRequestError("league and season are required", nil)
		}
		ctx, cancel := context.WithTimeout(e.Request.Context(), 45*time.Second)
		defer cancel()
		d, err := fetchSeason(ctx, key, leagueID, season)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		p := Derive(d.league, d.season, d.fixtures, d.teams, d.standings)
		// Slug collision hint so the admin can fix it before importing.
		if _, err := tournaments.BySlug(app, p.Slug); err == nil {
			p.Warnings = append(p.Warnings, fmt.Sprintf("slug %q already exists — pick another", p.Slug))
		}
		return e.JSON(http.StatusOK, p)
	})

	// POST /api/admin/tournaments/import — body: the (edited) Proposal plus
	// optional "competition" (id) and "status". Creates a draft and seeds it.
	se.Router.POST("/api/admin/tournaments/import", func(e *core.RequestEvent) error {
		key, err := needKey(e)
		if err != nil {
			return err
		}
		var body struct {
			Proposal
			Competition string `json:"competition"`
			Status      string `json:"status"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if body.LeagueID == 0 || body.Season == 0 {
			return apis.NewBadRequestError("leagueId and season are required", nil)
		}
		if _, err := tournaments.BySlug(app, body.Slug); err == nil {
			return apis.NewBadRequestError("slug already exists", nil)
		}
		ctx, cancel := context.WithTimeout(e.Request.Context(), 45*time.Second)
		defer cancel()
		d, err := fetchSeason(ctx, key, body.LeagueID, body.Season)
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		if len(d.fixtures) == 0 {
			return apis.NewBadRequestError("no fixtures published for this season yet", nil)
		}
		// Team metadata (codes / groups / iso2) comes from a fresh derivation
		// over the same cached data — the admin edits identity + structure,
		// not the seed rows.
		derived := Derive(d.league, d.season, d.fixtures, d.teams, d.standings)

		structure, _ := json.Marshal(body.Structure)
		syncJSON, _ := json.Marshal(body.Sync)
		spec, _ := json.Marshal(body.Forecast)
		pl := &tournaments.Payload{
			Slug: &body.Slug, Name: &body.Name, ShortName: &body.ShortName,
			StartsAt: &body.StartsAt, EndsAt: &body.EndsAt,
			Structure: structure, Sync: syncJSON, ForecastSpec: spec,
			ExtIDPrefix: &body.ExtIDPrefix,
		}
		if body.Competition != "" {
			pl.Competition = &body.Competition
		}
		if body.Status != "" {
			pl.Status = &body.Status
		}

		var rec *core.Record
		err = app.RunInTransaction(func(tx core.App) error {
			var err error
			rec, err = tournaments.Create(tx, pl)
			if err != nil {
				return err
			}
			return seedFromFixtures(tx, rec, derived, d.fixtures)
		})
		if err != nil {
			return err
		}
		nTeams, _ := app.CountRecords("teams", dbx.HashExp{"tournament": rec.Id})
		nMatches, _ := app.CountRecords("matches", dbx.HashExp{"tournament": rec.Id})
		return e.JSON(http.StatusOK, map[string]any{
			"id": rec.Id, "slug": rec.GetString("slug"), "teams": nTeams, "matches": nMatches,
		})
	}).Bind(apis.RequireAuth()).BindFunc(adminOnly)
}

// codeFor picks a 3-char team code: the provider's, else the first letters
// of the name; de-duplicated within the tournament (fifaCode is unique per
// tournament).
func codeFor(name, code string, used map[string]bool) string {
	c := strings.ToUpper(strings.TrimSpace(code))
	if len(c) > 3 || c == "" {
		letters := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				return r
			}
			return -1
		}, name)
		c = strings.ToUpper(letters)
		if len(c) > 3 {
			c = c[:3]
		}
		if c == "" {
			c = "TM"
		}
	}
	base := c
	for i := 0; used[c]; i++ {
		// Replace the last char with a digit to keep within 3 chars.
		if len(base) >= 3 {
			c = base[:2] + strconv.Itoa(i%10)
		} else {
			c = base + strconv.Itoa(i%10)
		}
		if i > 20 {
			break
		}
	}
	used[c] = true
	return c
}

// seedFromFixtures writes teams, groups and matches for a freshly created
// tournament. Mirrors seed.SeedTournament's row conventions so the rest of
// the app (standings, bracket resolution, live sync) treats imported
// tournaments exactly like the embedded WC2026 seed.
func seedFromFixtures(app core.App, t *core.Record, p *Proposal, fixtures []football.Fixture) error {
	st, err := tournaments.StructureOf(t)
	if err != nil {
		return fmt.Errorf("structure: %w", err)
	}
	valid := map[string]bool{}
	for _, c := range st.StageCodes() {
		valid[c] = true
	}
	teamsCol, err := app.FindCollectionByNameOrId("teams")
	if err != nil {
		return err
	}
	groupsCol, err := app.FindCollectionByNameOrId("tournament_groups")
	if err != nil {
		return err
	}
	matchesCol, err := app.FindCollectionByNameOrId("matches")
	if err != nil {
		return err
	}

	used := map[string]bool{}
	byProviderID := map[int]*core.Record{}
	groupOf := map[int]string{} // provider team id → group letter
	groupTeams := map[string][]string{}
	for _, tp := range p.Teams {
		groupOf[tp.ID] = tp.Group
		rec := core.NewRecord(teamsCol)
		rec.Set("tournament", t.Id)
		rec.Set("name", tp.Name)
		rec.Set("fifaCode", codeFor(tp.Name, tp.Code, used))
		rec.Set("iso2", tp.ISO2)
		if !tp.National {
			rec.Set("clubKey", Slugify(tp.Name))
		}
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("save team %s: %w", tp.Name, err)
		}
		byProviderID[tp.ID] = rec
		if tp.Group != "" {
			groupTeams[tp.Group] = append(groupTeams[tp.Group], rec.Id)
		}
	}
	for letter, ids := range groupTeams {
		rec := core.NewRecord(groupsCol)
		rec.Set("tournament", t.Id)
		rec.Set("letter", letter)
		rec.Set("teams", ids)
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("save group %s: %w", letter, err)
		}
	}

	// Stable order: kickoff, then provider id — `num` is what the bracket
	// resolver and UI use to order matches.
	sorted := make([]football.Fixture, len(fixtures))
	copy(sorted, fixtures)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].Date.Equal(sorted[j].Date) {
			return sorted[i].Date.Before(sorted[j].Date)
		}
		return sorted[i].ID < sorted[j].ID
	})
	prefix := t.GetString("extIdPrefix")
	for i, f := range sorted {
		kind, _ := classifyRound(f.Round)
		stage := "group"
		if kind == kindKnockout {
			stage = knockoutStage(f.Round).Code
		}
		if !valid[stage] {
			return fmt.Errorf("round %q maps to stage %q which is not in the structure", f.Round, stage)
		}
		rec := core.NewRecord(matchesCol)
		rec.Set("tournament", t.Id)
		rec.Set("extId", fmt.Sprintf("%s-AF-%d", prefix, f.ID))
		rec.Set("stage", stage)
		rec.Set("num", i+1)
		rec.Set("roundLabel", f.Round)
		rec.Set("kickoff", f.Date.UTC())
		rec.Set("status", "scheduled")
		if stage == "group" {
			// Group membership was derived per team (round letters, standings
			// or schedule components) — both sides share it.
			letter := groupOf[f.HomeID]
			if letter == "" {
				letter = groupOf[f.AwayID]
			}
			if letter == "" {
				letter = "A"
			}
			rec.Set("groupLetter", letter)
		}
		home, hok := byProviderID[f.HomeID]
		away, aok := byProviderID[f.AwayID]
		if hok {
			rec.Set("homeTeam", home.Id)
		} else {
			rec.Set("homeLabel", nz(f.HomeName, "TBD"))
		}
		if aok {
			rec.Set("awayTeam", away.Id)
		} else {
			rec.Set("awayLabel", nz(f.AwayName, "TBD"))
		}
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("save match %s: %w", rec.GetString("extId"), err)
		}
	}
	return nil
}

func nz(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
