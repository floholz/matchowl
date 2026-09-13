// Package sync keeps the matches collection up to date: a cron job pulls
// results for every active tournament from that tournament's configured
// provider (API-Football or openfootball), a superuser endpoint forces a
// refresh, and another superuser endpoint applies manual results when the
// provider is wrong or no API key is configured.
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/football"
	"github.com/floholz/matchowl/internal/importer"
	"github.com/floholz/matchowl/internal/tournaments"
	"github.com/floholz/matchowl/internal/users"
)

// syncMetaPrefix + tournament slug is the app_meta row that stores the last
// results-sync outcome for that tournament, surfaced on the admin dashboard.
const syncMetaPrefix = "results_sync:"

// recordSyncStatus persists the outcome of a sync run (best effort) so the
// admin dashboard can show when results last updated and whether it succeeded.
func recordSyncStatus(app core.App, slug, source string, updated int, runErr error) {
	col, err := app.FindCollectionByNameOrId("app_meta")
	if err != nil {
		return
	}
	key := syncMetaPrefix + slug
	rec, err := app.FindFirstRecordByFilter("app_meta",
		"key = {:k}", map[string]any{"k": key})
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("key", key)
	}
	val := map[string]any{
		"at":         time.Now().UTC().Format(time.RFC3339),
		"tournament": slug,
		"source":     source,
		"updated":    updated,
		"ok":         runErr == nil,
	}
	if runErr != nil {
		val["error"] = runErr.Error()
	}
	rec.Set("value", val)
	if err := app.Save(rec); err != nil {
		log.Printf("[sync] record status: %v", err)
	}
}

// readSyncStatus returns the last recorded sync outcome per tournament slug.
func readSyncStatus(app core.App) map[string]any {
	recs, err := app.FindRecordsByFilter("app_meta",
		"key ~ {:p}", "", 0, 0, map[string]any{"p": syncMetaPrefix + "%"})
	if err != nil {
		return nil
	}
	out := map[string]any{}
	for _, rec := range recs {
		var v map[string]any
		if err := rec.UnmarshalJSONField("value", &v); err == nil {
			out[rec.GetString("key")[len(syncMetaPrefix):]] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// cronExpr is the default sync cadence: every 30 minutes => max 48 requests/day
// per active tournament, comfortably under the API-Football free tier
// (100/day). Override at runtime with the SYNC_CRON env var (see Register).
const cronExpr = "*/30 * * * *"

// nameAliases maps API-Football names that differ from the openfootball seed
// names to the seeded team name.
var nameAliases = map[string]string{
	football.NormalizeName("Korea Republic"):     football.NormalizeName("South Korea"),
	football.NormalizeName("Czechia"):            football.NormalizeName("Czech Republic"),
	football.NormalizeName("USA"):                football.NormalizeName("United States"),
	football.NormalizeName("IR Iran"):            football.NormalizeName("Iran"),
	football.NormalizeName("Türkiye"):            football.NormalizeName("Turkey"),
	football.NormalizeName("Turkiye"):            football.NormalizeName("Turkey"), // ü-less, in case the API sends a folded form
	football.NormalizeName("Cape Verde Islands"): football.NormalizeName("Cape Verde"),
	football.NormalizeName("Congo DR"):           football.NormalizeName("DR Congo"),
}

func canonName(s string) string {
	n := football.NormalizeName(s)
	if a, ok := nameAliases[n]; ok {
		return a
	}
	return n
}

// runner is a tournament's resolved live-results source.
type runner struct {
	slug   string
	source string
	run    func(context.Context) (int, error)
}

// pickProvider resolves a tournament's live-results source from its sync
// config. RESULTS_SOURCE=apifootball|openfootball still forces the choice
// globally. Returns a label and a sync function; when no source resolves the
// function is nil and reason explains why (surfaced on the admin sync card).
func pickProvider(app core.App, t *core.Record) (source string, run func(context.Context) (int, error), reason string) {
	cfg, err := tournaments.SyncOf(t)
	if err != nil {
		log.Printf("[sync] %s: bad sync config: %v", t.GetString("slug"), err)
		return "", nil, "invalid sync config: " + err.Error()
	}
	key := os.Getenv("API_FOOTBALL_KEY")
	mode := os.Getenv("RESULTS_SOURCE")

	apiFn := func(ctx context.Context) (int, error) {
		return SyncOnce(ctx, app, football.New(key, cfg.APIFootballLeague, cfg.Season), t)
	}
	ofFn := func(ctx context.Context) (int, error) {
		return openfootballSync(ctx, app, t, cfg.OpenfootballURL)
	}

	provider := cfg.Provider
	if mode == "openfootball" {
		provider = tournaments.ProviderOpenfootball
	} else if mode == "apifootball" {
		provider = tournaments.ProviderAPIFootball
	}

	forced := ""
	if mode != "" {
		forced = " (forced by RESULTS_SOURCE=" + mode + ")"
	}

	switch provider {
	case tournaments.ProviderManual:
		return "", nil, "sync provider is “manual” — results are entered by hand"
	case tournaments.ProviderOpenfootball:
		if cfg.OpenfootballURL == "" {
			return "", nil, "provider is openfootball" + forced + " but the tournament has no openfootballURL"
		}
		return "openfootball", ofFn, ""
	case tournaments.ProviderAPIFootball:
		if key == "" {
			return "", nil, "provider is api-football" + forced + " but API_FOOTBALL_KEY is not set"
		}
		return "api-football", apiFn, ""
	default: // auto: prefer API-Football only if the key can actually fetch the season.
		var apiNote string
		if key != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			fx, err := football.New(key, cfg.APIFootballLeague, cfg.Season).Fixtures(ctx)
			if err == nil && len(fx) > 0 {
				return "api-football", apiFn, ""
			}
			log.Printf("[sync] %s: API-Football key can't reach season %d (free plan?) — using openfootball",
				t.GetString("slug"), cfg.Season)
			apiNote = fmt.Sprintf("API-Football key can't reach pool %d season %d (paid plan needed?)",
				cfg.APIFootballLeague, cfg.Season)
		} else {
			apiNote = "API_FOOTBALL_KEY is not set"
		}
		if cfg.OpenfootballURL == "" {
			return "", nil, "provider is auto: " + apiNote + ", and the tournament has no openfootballURL fallback"
		}
		return "openfootball", ofFn, ""
	}
}

// skipped is an active tournament that has no usable results source.
type skipped struct {
	slug   string
	reason string
}

// resolveRunners splits the active tournaments into runners (source resolved)
// and skipped ones (with the reason no source resolved).
func resolveRunners(app core.App) ([]runner, []skipped) {
	recs, err := tournaments.Active(app)
	if err != nil {
		log.Printf("[sync] list active tournaments: %v", err)
		return nil, nil
	}
	var rs []runner
	var sk []skipped
	for _, t := range recs {
		source, run, reason := pickProvider(app, t)
		if run == nil {
			sk = append(sk, skipped{slug: t.GetString("slug"), reason: reason})
			continue
		}
		rs = append(rs, runner{slug: t.GetString("slug"), source: source, run: run})
	}
	return rs, sk
}

// noSourceReason explains an empty runner list for the admin card: either
// nothing is active (list what exists and its status), or every active
// tournament was skipped (their reasons are listed separately).
func noSourceReason(app core.App, sk []skipped) string {
	if len(sk) > 0 {
		return "no active tournament has a usable results source"
	}
	all, err := tournaments.All(app)
	if err != nil || len(all) == 0 {
		return "no tournaments exist yet — create one via the admin API"
	}
	parts := make([]string, 0, len(all))
	for _, t := range all {
		parts = append(parts, t.GetString("slug")+" ("+t.GetString("status")+")")
	}
	return "no tournament is “active” — sync only runs for active tournaments; found: " + strings.Join(parts, ", ")
}

// activeRunners resolves the sync source for every active tournament.
func activeRunners(app core.App) []runner {
	rs, _ := resolveRunners(app)
	return rs
}

// runAll executes every active tournament's sync and records each outcome.
// Returns the total number of updated match records and the first error.
func runAll(ctx context.Context, app core.App, rs []runner) (int, error) {
	total := 0
	var firstErr error
	for _, r := range rs {
		n, err := r.run(ctx)
		recordSyncStatus(app, r.slug, r.source, n, err)
		total += n
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return total, firstErr
}

// Register wires the live-results cron + manual override endpoints.
// Called from the OnServe hook.
func Register(app core.App, se *core.ServeEvent) {
	// SYNC_CRON overrides the default cadence without a rebuild — e.g. tighten
	// to "*/5 * * * *" for near-instant scores during matches.
	expr := os.Getenv("SYNC_CRON")
	if expr == "" {
		expr = cronExpr
	}

	// The provider probe runs per cron tick (not once at boot) so activating
	// a tournament or fixing a key doesn't need a restart.
	app.Cron().MustAdd("results-sync", expr, func() {
		rs := activeRunners(app)
		if len(rs) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if _, err := runAll(ctx, app, rs); err != nil {
			log.Printf("[sync] %v", err)
		}
	})
	log.Printf("[sync] auto-sync cron registered (%s) — sources resolve per active tournament", expr)

	// Force a sync now (superuser).
	se.Router.POST("/api/sync/refresh", func(e *core.RequestEvent) error {
		rs := activeRunners(app)
		if len(rs) == 0 {
			return e.JSON(400, map[string]string{"error": "no active tournament with a results source"})
		}
		ctx, cancel := context.WithTimeout(e.Request.Context(), 60*time.Second)
		defer cancel()
		n, err := runAll(ctx, app, rs)
		if err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		return e.JSON(200, map[string]any{"status": "ok", "updated": n})
	}).Bind(apis.RequireSuperuserAuth())

	// Admin-gated sync dashboard: live status + a manual "sync now" button.
	sg := se.Router.Group("/api/admin/sync")
	sg.Bind(apis.RequireAuth())
	sg.BindFunc(func(e *core.RequestEvent) error {
		if e.Auth == nil || !users.IsAdmin(e.Auth) {
			return apis.NewForbiddenError("admin only", nil)
		}
		return e.Next()
	})

	// GET /api/admin/sync/status — per-tournament source, cadence, last runs,
	// and (when any source is API-Football) the plan + request quota.
	sg.GET("/status", func(e *core.RequestEvent) error {
		rs, sk := resolveRunners(app)
		sources := make([]map[string]any, 0, len(rs))
		hasAPI := false
		for _, r := range rs {
			sources = append(sources, map[string]any{"tournament": r.slug, "source": r.source})
			if r.source == "api-football" {
				hasAPI = true
			}
		}
		skippedOut := make([]map[string]any, 0, len(sk))
		for _, s := range sk {
			skippedOut = append(skippedOut, map[string]any{"tournament": s.slug, "reason": s.reason})
		}
		out := map[string]any{
			"sources":  sources,
			"skipped":  skippedOut,
			"autoSync": len(rs) > 0,
			"cron":     expr,
			"lastRun":  readSyncStatus(app),
		}
		if len(rs) == 0 {
			out["reason"] = noSourceReason(app, sk)
		}
		if hasAPI {
			if key := os.Getenv("API_FOOTBALL_KEY"); key != "" {
				ctx, cancel := context.WithTimeout(e.Request.Context(), 12*time.Second)
				defer cancel()
				if st, err := football.New(key, 0, 0).Status(ctx); err == nil {
					out["account"] = st
				} else {
					out["accountError"] = err.Error()
				}
			}
		}
		return e.JSON(200, out)
	})

	// POST /api/admin/sync/run — force a sync now and return the outcome.
	sg.POST("/run", func(e *core.RequestEvent) error {
		rs := activeRunners(app)
		if len(rs) == 0 {
			return e.JSON(400, map[string]string{"error": "no active tournament with a results source"})
		}
		ctx, cancel := context.WithTimeout(e.Request.Context(), 60*time.Second)
		defer cancel()
		n, err := runAll(ctx, app, rs)
		if err != nil {
			return e.JSON(500, map[string]any{"error": err.Error(), "lastRun": readSyncStatus(app)})
		}
		return e.JSON(200, map[string]any{"status": "ok", "updated": n, "lastRun": readSyncStatus(app)})
	})

	// Manual result override (superuser). Body: ftHome,ftAway,etHome,etAway,
	// penHome,penAway (ints, et/pen optional) and status.
	se.Router.POST("/api/admin/matches/{id}/result", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		rec, err := app.FindRecordById("matches", id)
		if err != nil {
			return e.JSON(404, map[string]string{"error": "match not found"})
		}
		var body struct {
			FTHome, FTAway   *int
			ETHome, ETAway   *int
			PenHome, PenAway *int
			Status           string
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(400, map[string]string{"error": err.Error()})
		}
		applyResult(rec, isKnockout(app, rec), body.Status, body.FTHome, body.FTAway, body.ETHome, body.ETAway, body.PenHome, body.PenAway)
		if err := app.Save(rec); err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		if trec, err := app.FindRecordById("tournaments", rec.GetString("tournament")); err == nil {
			if st, err := tournaments.StructureOf(trec); err == nil {
				FixTwoLeggedAdvancers(app, trec.Id, st)
			}
		}
		if err := ResolveBracket(app); err != nil {
			log.Printf("[sync] resolve after manual override: %v", err)
		}
		return e.JSON(200, map[string]any{"status": "ok", "id": rec.Id})
	}).Bind(apis.RequireSuperuserAuth())
}

// SyncOnce pulls one tournament's fixtures once and updates matched records,
// returning how many records changed.
func SyncOnce(ctx context.Context, app core.App, client *football.Client, t *core.Record) (int, error) {
	fixtures, err := client.Fixtures(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetch fixtures: %w", err)
	}

	st, err := tournaments.StructureOf(t)
	if err != nil {
		return 0, fmt.Errorf("structure: %w", err)
	}

	matches, err := app.FindRecordsByFilter("matches",
		"tournament = {:t}", "kickoff", 0, 0, map[string]any{"t": t.Id})
	if err != nil {
		return 0, fmt.Errorf("load matches: %w", err)
	}

	// Index our matches by the normalized team-name pair (group stage) so we
	// can line them up with provider fixtures regardless of fixture ids.
	teamName := map[string]string{} // teamId -> normalized name
	teams, _ := app.FindRecordsByFilter("teams",
		"tournament = {:t}", "", 0, 0, map[string]any{"t": t.Id})
	for _, tm := range teams {
		teamName[tm.Id] = canonName(tm.GetString("name"))
	}

	byPair := map[string]*core.Record{}
	byExt := map[string]*core.Record{} // imported tournaments: "<prefix>-AF-<fixtureId>"
	maxNum := 0
	for _, mrec := range matches {
		h := teamName[mrec.GetString("homeTeam")]
		a := teamName[mrec.GetString("awayTeam")]
		if h != "" && a != "" {
			byPair[h+"|"+a] = mrec
		}
		if ext := mrec.GetString("extId"); strings.Contains(ext, "-AF-") {
			byExt[ext] = mrec
		}
		if n := mrec.GetInt("num"); n > maxNum {
			maxNum = n
		}
	}
	teamByName := map[string]string{} // normalized name -> teamId
	for id, n := range teamName {
		teamByName[n] = id
	}
	prefix := t.GetString("extIdPrefix")

	// Qualifying rounds were never seeded (see importer.QualifierRounds); a
	// qualifier between two teams that both reached the main stage would
	// otherwise pair-match a seeded match and smear its result onto it.
	excluded := importer.QualifierRounds(fixtures)

	// Kickoff order so backfilled matches get `num`s in play order.
	sort.SliceStable(fixtures, func(i, j int) bool {
		if !fixtures[i].Date.Equal(fixtures[j].Date) {
			return fixtures[i].Date.Before(fixtures[j].Date)
		}
		return fixtures[i].ID < fixtures[j].ID
	})

	updated := 0
	for _, f := range fixtures {
		if excluded[f.Round] {
			continue
		}
		key := canonName(f.HomeName) + "|" + canonName(f.AwayName)
		rec, ok := byPair[key]
		if !ok {
			// Imported (API-Football-seeded) knockout rows are keyed by the
			// provider fixture id; once the provider knows both teams, fill
			// them in so the tip opens and the result below applies.
			ext := fmt.Sprintf("%s-AF-%d", prefix, f.ID)
			rec, ok = byExt[ext]
			if !ok && len(byExt) > 0 {
				// An imported tournament met a fixture the importer never saw:
				// UEFA-style competitions create knockout fixtures one draw at
				// a time, so backfill the match (and its stage) now.
				rec, err = backfillMatch(app, t, st, f, ext, maxNum+1, teamByName)
				if err != nil {
					log.Printf("[sync] %s: backfill %q: %v", t.GetString("slug"), f.Round, err)
					continue
				}
				maxNum++
				byExt[ext] = rec
				updated++
				ok = true
			}
			if !ok {
				// Otherwise KO matches resolve via ResolveBracket; unmatched
				// group names usually mean an alias is missing — not fatal.
				continue
			}
			hID, aID := teamByName[canonName(f.HomeName)], teamByName[canonName(f.AwayName)]
			changed := false
			if rec.GetString("homeTeam") == "" && hID != "" {
				rec.Set("homeTeam", hID)
				rec.Set("homeLabel", "")
				changed = true
			}
			if rec.GetString("awayTeam") == "" && aID != "" {
				rec.Set("awayTeam", aID)
				rec.Set("awayLabel", "")
				changed = true
			}
			if rec.GetString("homeTeam") == "" || rec.GetString("awayTeam") == "" {
				if changed && app.Save(rec) == nil {
					updated++
				}
				continue
			}
		}
		status := "scheduled"
		switch {
		case f.Finished():
			status = "finished"
		case f.Live():
			status = "live"
		}
		// API `score.extratime` is the ET-only delta; our model (and Tips /
		// scoring) use the cumulative after-120 score, which is exactly the
		// provider `goals` field once a match has gone to extra time.
		var etH, etA *int
		if f.ETHome != nil || f.ETAway != nil {
			etH, etA = f.HomeGoals, f.AwayGoals
		}
		// During play the provider reports the current score only in `goals`
		// (`score.fulltime` stays null until FT) — persist it as ftHome/ftAway
		// so the app can show live scores.
		ftH, ftA := f.FTHome, f.FTAway
		if status == "live" && ftH == nil && ftA == nil {
			ftH, ftA = f.HomeGoals, f.AwayGoals
		}
		// Schedule changes: leagues confirm exact kick-off times round by
		// round (the provider first lists a whole round on one placeholder
		// day), and fixtures get postponed. Follow the provider for anything
		// not yet played so tips lock at the real time.
		sched := applySchedule(rec, f, status)
		// Skip if nothing changed (avoids needless recompute storms: every
		// save of a finished or knockout match triggers a full score rebuild).
		// A finished match without finalizedAt (e.g. hand-edited in the admin
		// UI) still saves, so it gets finalized and scored.
		if !sched && rec.GetString("status") == status &&
			(status != "finished" || rec.GetString("finalizedAt") != "") &&
			(ftH == nil || rec.GetInt("ftHome") == *ftH) &&
			(ftA == nil || rec.GetInt("ftAway") == *ftA) &&
			rec.GetInt("etHome") == ip(etH) && rec.GetInt("etAway") == ip(etA) &&
			rec.GetInt("penHome") == ip(f.PenHome) && rec.GetInt("penAway") == ip(f.PenAway) {
			continue
		}
		applyResult(rec, st.IsKnockout(rec.GetString("stage")), status, ftH, ftA, etH, etA, f.PenHome, f.PenAway)
		if app.Save(rec) == nil {
			updated++
		}
	}

	updated += FixTwoLeggedAdvancers(app, t.Id, st)

	if err := ResolveBracket(app); err != nil {
		log.Printf("[sync] resolve bracket: %v", err)
	}
	log.Printf("[sync] %s: fixtures=%d updated=%d", t.GetString("slug"), len(fixtures), updated)
	return updated, nil
}

// appendStage adds the stage to the structure if it isn't there yet,
// reporting whether it changed the structure. A group/table round whose
// structure lacks a group stage is refused rather than invented.
func appendStage(st *tournaments.Structure, stage tournaments.Stage) (bool, error) {
	if st.Stage(stage.Code) != nil {
		return false, nil
	}
	if stage.Kind != tournaments.KindKnockout {
		return false, fmt.Errorf("no %q stage in the structure", stage.Code)
	}
	st.Stages = append(st.Stages, stage)
	if err := st.Validate(); err != nil {
		st.Stages = st.Stages[:len(st.Stages)-1]
		return false, err
	}
	return true, nil
}

// backfillMatch creates a match row for a provider fixture that appeared
// after the tournament was imported — UEFA-style competitions publish
// knockout fixtures one draw at a time, so the importer never saw them.
// New knockout stages are appended to the structure as their draws happen.
func backfillMatch(app core.App, t *core.Record, st *tournaments.Structure, f football.Fixture, extID string, num int, teamByName map[string]string) (*core.Record, error) {
	stage := importer.StageFor(f.Round)
	changed, err := appendStage(st, stage)
	if err != nil {
		return nil, err
	}
	if changed {
		raw, err := json.Marshal(st)
		if err != nil {
			return nil, err
		}
		t.Set("structure", string(raw))
	}
	// The import only saw the fixtures published at the time; matches from
	// later draws can extend the season.
	if end := t.GetDateTime("endsAt"); !end.IsZero() && f.Date.After(end.Time()) {
		t.Set("endsAt", f.Date.UTC())
		changed = true
	}
	if changed {
		if err := app.Save(t); err != nil {
			return nil, fmt.Errorf("save tournament: %w", err)
		}
	}
	col, err := app.FindCollectionByNameOrId("matches")
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(col)
	rec.Set("tournament", t.Id)
	rec.Set("extId", extID)
	rec.Set("stage", stage.Code)
	rec.Set("num", num)
	rec.Set("roundLabel", f.Round)
	rec.Set("kickoff", f.Date.UTC())
	rec.Set("status", "scheduled")
	hID, aID := teamByName[canonName(f.HomeName)], teamByName[canonName(f.AwayName)]
	if hID != "" {
		rec.Set("homeTeam", hID)
	} else if strings.TrimSpace(f.HomeName) != "" {
		rec.Set("homeLabel", f.HomeName)
	} else {
		rec.Set("homeLabel", "TBD")
	}
	if aID != "" {
		rec.Set("awayTeam", aID)
	} else if strings.TrimSpace(f.AwayName) != "" {
		rec.Set("awayLabel", f.AwayName)
	} else {
		rec.Set("awayLabel", "TBD")
	}
	if stage.Kind == tournaments.KindGroup {
		rec.Set("groupLetter", groupLetterFor(app, t.Id, hID, aID))
	}
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("save match %s: %w", extID, err)
	}
	return rec, nil
}

// groupLetterFor finds the group either team belongs to. Backfilled group
// matches are rare (a rescheduled fixture under a new provider id), so this
// queries per call instead of pre-building a map.
func groupLetterFor(app core.App, tournamentID, homeID, awayID string) string {
	for _, id := range []string{homeID, awayID} {
		if id == "" {
			continue
		}
		g, err := app.FindFirstRecordByFilter("tournament_groups",
			"tournament = {:t} && teams ~ {:id}", map[string]any{"t": tournamentID, "id": id})
		if err == nil {
			return g.GetString("letter")
		}
	}
	return "A"
}

// APICheck is a dev diagnostic: fetch a season's fixtures from API-Football
// and report parse health, team-name mapping coverage against our seed, how
// many of our match rows resolve, and the status / ET / penalty distribution
// (point it at a finished season like 2022 to validate the results path).
func APICheck(ctx context.Context, app core.App, client *football.Client, yr int) (map[string]any, error) {
	fixtures, err := client.FixturesForSeason(ctx, yr)
	if err != nil {
		return nil, err
	}

	teams, _ := app.FindRecordsByFilter("teams", "id != ''", "", 0, 0)
	seedCanon := map[string]string{} // canonName -> seeded display name
	teamName := map[string]string{}  // teamId -> canonName
	for _, t := range teams {
		c := canonName(t.GetString("name"))
		seedCanon[c] = t.GetString("name")
		teamName[t.Id] = c
	}

	matches, _ := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 0, 0)
	byPair := map[string]*core.Record{}
	for _, m := range matches {
		h, a := teamName[m.GetString("homeTeam")], teamName[m.GetString("awayTeam")]
		if h != "" && a != "" {
			byPair[h+"|"+a] = m
		}
	}

	statusHist := map[string]int{}
	unmapped := map[string]bool{}
	matchedRows := map[string]bool{}
	etCount, penCount := 0, 0
	var sample []map[string]any

	for _, f := range fixtures {
		statusHist[f.Status]++
		for _, nm := range []string{f.HomeName, f.AwayName} {
			if _, ok := seedCanon[canonName(nm)]; !ok {
				unmapped[nm] = true
			}
		}
		if rec, ok := byPair[canonName(f.HomeName)+"|"+canonName(f.AwayName)]; ok {
			matchedRows[rec.Id] = true
		}
		if f.ETHome != nil || f.ETAway != nil {
			etCount++
		}
		if f.PenHome != nil || f.PenAway != nil {
			penCount++
		}
		// Prefer extra-time / penalty fixtures in the sample — that's the
		// path most worth eyeballing.
		if (f.ETHome != nil || f.PenHome != nil) && len(sample) < 6 {
			sample = append(sample, map[string]any{
				"round": f.Round, "status": f.Status,
				"home": f.HomeName, "away": f.AwayName,
				"ft":                []any{f.FTHome, f.FTAway},
				"et":                []any{f.ETHome, f.ETAway},
				"pen":               []any{f.PenHome, f.PenAway},
				"advancerDerivable": f.Finished(),
			})
		}
	}
	unm := make([]string, 0, len(unmapped))
	for n := range unmapped {
		unm = append(unm, n)
	}
	sort.Strings(unm)

	return map[string]any{
		"season":           yr,
		"fixtures":         len(fixtures),
		"statusHistogram":  statusHist,
		"unmappedTeams":    unm,
		"ourMatchesTotal":  len(matches),
		"ourMatchesMapped": len(matchedRows),
		"withExtraTime":    etCount,
		"withPenalties":    penCount,
		"sample":           sample,
	}, nil
}

// applySchedule moves a match to the provider's kick-off time and round
// label, reporting whether anything changed. Finished matches follow too:
// the provider's date is the truth, and a match that was played on a
// different day than we had it (Lazio – Milan, 2026-09-12: brought forward
// a day, seen only after it had finished) must not sit in the feed under
// the wrong day.
func applySchedule(rec *core.Record, f football.Fixture, status string) bool {
	if f.Date.IsZero() {
		return false
	}
	changed := false
	want := f.Date.UTC().Truncate(time.Second)
	if have := rec.GetDateTime("kickoff").Time().UTC().Truncate(time.Second); !have.Equal(want) {
		rec.Set("kickoff", want)
		changed = true
	}
	if f.Round != "" && rec.GetString("roundLabel") != f.Round {
		rec.Set("roundLabel", f.Round)
		changed = true
	}
	return changed
}

func ip(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

// isKnockout reports whether a match's stage is a knockout stage per its
// tournament's structure (fallback for unreadable structures: anything not
// literally "group").
func isKnockout(app core.App, rec *core.Record) bool {
	stage := rec.GetString("stage")
	t, err := app.FindRecordById("tournaments", rec.GetString("tournament"))
	if err != nil {
		return stage != "group"
	}
	st, err := tournaments.StructureOf(t)
	if err != nil {
		return stage != "group"
	}
	return st.IsKnockout(stage)
}

// ApplyResult is the exported entry point (used by the dev simulator) that
// writes a result onto a match record using the same logic as live sync /
// manual override. It resolves the knockout-ness of the match from its
// tournament's structure.
func ApplyResult(app core.App, rec *core.Record, status string, ftH, ftA, etH, etA, penH, penA *int) {
	applyResult(rec, isKnockout(app, rec), status, ftH, ftA, etH, etA, penH, penA)
}

// applyResult writes scores/status onto a match record and, for knockout
// matches, derives the advancer (penalties > ET > regulation).
func applyResult(rec *core.Record, knockout bool, status string, ftH, ftA, etH, etA, penH, penA *int) {
	if status != "" {
		rec.Set("status", status)
	}
	if ftH != nil {
		rec.Set("ftHome", *ftH)
	}
	if ftA != nil {
		rec.Set("ftAway", *ftA)
	}
	rec.Set("etHome", ip(etH))
	rec.Set("etAway", ip(etA))
	rec.Set("penHome", ip(penH))
	rec.Set("penAway", ip(penA))

	finished := rec.GetString("status") == "finished"
	if finished {
		rec.Set("finalizedAt", time.Now().UTC())
	}

	if !knockout || !finished {
		return
	}
	// Knockout advancer resolution.
	home := rec.GetString("homeTeam")
	away := rec.GetString("awayTeam")
	switch {
	case penH != nil && penA != nil && *penH != *penA:
		if *penH > *penA {
			rec.Set("penWinner", home)
			rec.Set("advancer", home)
		} else {
			rec.Set("penWinner", away)
			rec.Set("advancer", away)
		}
	case etH != nil && etA != nil && *etH != *etA:
		if *etH > *etA {
			rec.Set("advancer", home)
		} else {
			rec.Set("advancer", away)
		}
	case ftH != nil && ftA != nil && *ftH != *ftA:
		if *ftH > *ftA {
			rec.Set("advancer", home)
		} else {
			rec.Set("advancer", away)
		}
	}
}
