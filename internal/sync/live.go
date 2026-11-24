package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	gosync "sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/football"
	"github.com/floholz/matchowl/internal/tournaments"
)

// Kick-off-aware sync (PLAN.md, "pools at the center", phase 4). Instead of
// pulling every season on a flat cron, a pass runs every minute and spends
// requests only around kick-offs:
//
//   - nothing due: no request at all;
//   - matches live or due: one /fixtures?live=<league ids> request for all
//     covered leagues together (it carries each match's events);
//   - matches that left the live answer (just finished, or a delayed or
//     postponed start): /fixtures?ids= in batches of 20, every minute for
//     the ones that were live, every ten minutes for late starters;
//   - rows the poller can't map yet (no provider id): that season's full
//     sync, at most every ten minutes.
//
// The full schedule refresh per season (kick-off changes, new knockout
// fixtures) runs twice a day on its own cron (fullCronExpr).

const (
	// liveCronExpr runs the live pass.
	liveCronExpr = "* * * * *"
	// fullCronExpr refreshes every season's whole schedule.
	fullCronExpr = "15 5,17 * * *"
	// dueBefore/dueAfter bound the window a scheduled match counts as due.
	dueBefore = time.Minute
	dueAfter  = 4 * time.Hour
	// lateEvery throttles the by-id checks of matches past kick-off that
	// never showed up live, and the full-sync fallback.
	lateEvery = 10 * time.Minute
)

// liveSeason is one active API-Football season the poller covers.
type liveSeason struct {
	t      *core.Record
	st     *tournaments.Structure
	league int
	season int
}

var (
	liveMu       gosync.Mutex
	lateChecked  = map[string]time.Time{} // match id → last by-id check
	fullFallback = map[string]time.Time{} // tournament id → last full sync from the poller
)

// apiSeasons lists the active seasons synced from API-Football.
func apiSeasons(app core.App) []liveSeason {
	if os.Getenv("API_FOOTBALL_KEY") == "" || os.Getenv("RESULTS_SOURCE") == "openfootball" {
		return nil
	}
	recs, err := tournaments.Active(app)
	if err != nil {
		return nil
	}
	var out []liveSeason
	for _, t := range recs {
		cfg, err := tournaments.SyncOf(t)
		if err != nil || cfg.APIFootballLeague == 0 || cfg.Season == 0 {
			continue
		}
		switch cfg.Provider {
		case tournaments.ProviderManual, tournaments.ProviderOpenfootball:
			continue
		}
		st, err := tournaments.StructureOf(t)
		if err != nil {
			continue
		}
		out = append(out, liveSeason{t: t, st: st, league: cfg.APIFootballLeague, season: cfg.Season})
	}
	return out
}

// LiveResult summarises one live pass (for the log and the admin card).
type LiveResult struct {
	Due      int `json:"due"`
	Requests int `json:"requests"`
	Updated  int `json:"updated"`
}

// LivePass runs one kick-off-aware pass.
func LivePass(ctx context.Context, app core.App) (LiveResult, error) {
	liveMu.Lock()
	defer liveMu.Unlock()
	res := LiveResult{}
	seasons := apiSeasons(app)
	if len(seasons) == 0 {
		return res, nil
	}
	now := clock.Now(app)
	byTournament := map[string]*liveSeason{}
	tids := make([]any, 0, len(seasons))
	for i := range seasons {
		byTournament[seasons[i].t.Id] = &seasons[i]
		tids = append(tids, seasons[i].t.Id)
	}
	var due []*core.Record
	if err := app.RecordQuery("matches").
		AndWhere(dbx.In("tournament", tids...)).
		AndWhere(dbx.NewExp("(status = 'live' OR (status = 'scheduled' AND kickoff <= {:soon} AND kickoff >= {:since}))", dbx.Params{
			"soon":  now.Add(dueBefore).UTC().Format(types.DefaultDateLayout),
			"since": now.Add(-dueAfter).UTC().Format(types.DefaultDateLayout),
		})).
		All(&due); err != nil {
		return res, err
	}
	res.Due = len(due)
	if len(due) == 0 {
		return res, nil
	}
	key := os.Getenv("API_FOOTBALL_KEY")
	client := football.New(key, 0, 0)

	// One live request for every league with something due.
	leagueSet := map[int]bool{}
	for _, m := range due {
		if s := byTournament[m.GetString("tournament")]; s != nil {
			leagueSet[s.league] = true
		}
	}
	leagues := make([]int, 0, len(leagueSet))
	for l := range leagueSet {
		leagues = append(leagues, l)
	}
	live, err := client.Live(ctx, leagues)
	res.Requests++
	if err != nil {
		return res, fmt.Errorf("live: %w", err)
	}
	byAF := map[int]*core.Record{}
	for _, m := range due {
		if id := m.GetInt("afId"); id != 0 {
			byAF[id] = m
		}
	}
	touched := map[string]bool{}
	seen := map[string]bool{}
	for _, f := range live {
		rec := byAF[f.ID]
		if rec == nil {
			// Live but not in our window (kick-off moved earlier): look it up.
			if r, err := app.FindFirstRecordByFilter("matches", "afId = {:id}", map[string]any{"id": f.ID}); err == nil && byTournament[r.GetString("tournament")] != nil {
				rec = r
			}
		}
		if rec == nil {
			continue
		}
		seen[rec.Id] = true
		s := byTournament[rec.GetString("tournament")]
		if applyFixture(app, s.st, rec, f) {
			res.Updated++
			touched[rec.GetString("tournament")] = true
		}
	}

	// What the live answer didn't carry: finished, or late.
	var ids []int
	for _, m := range due {
		if seen[m.Id] {
			continue
		}
		id := m.GetInt("afId")
		if id == 0 {
			// Not mapped yet: the season's full sync sets afId.
			tid := m.GetString("tournament")
			if now.Sub(fullFallback[tid]) >= lateEvery {
				fullFallback[tid] = now
				s := byTournament[tid]
				n, err := SyncOnce(ctx, app, football.New(key, s.league, s.season), s.t)
				res.Requests++
				recordSyncStatus(app, s.t.GetString("slug"), "api-football", n, err)
				res.Updated += n
			}
			continue
		}
		wasLive := m.GetString("status") == "live"
		if !wasLive && now.Sub(lateChecked[m.Id]) < lateEvery {
			continue
		}
		if !wasLive && m.GetDateTime("kickoff").Time().After(now) {
			continue // due in the next minute, not late yet
		}
		lateChecked[m.Id] = now
		ids = append(ids, id)
	}
	for len(ids) > 0 {
		n := min(len(ids), football.MaxIDs)
		batch := ids[:n]
		ids = ids[n:]
		fx, err := client.ByIDs(ctx, batch)
		res.Requests++
		if err != nil {
			log.Printf("[sync] live: ids: %v", err)
			continue
		}
		for _, f := range fx {
			rec := byAF[f.ID]
			if rec == nil {
				continue
			}
			s := byTournament[rec.GetString("tournament")]
			if applyFixture(app, s.st, rec, f) {
				res.Updated++
				touched[rec.GetString("tournament")] = true
			}
		}
	}

	for tid := range touched {
		if s := byTournament[tid]; s != nil {
			res.Updated += FixTwoLeggedAdvancers(app, tid, s.st)
		}
	}
	if len(touched) > 0 {
		if err := ResolveBracket(app); err != nil {
			log.Printf("[sync] resolve bracket: %v", err)
		}
	}
	// Forget throttles of matches long gone.
	for id, at := range lateChecked {
		if now.Sub(at) > 2*dueAfter {
			delete(lateChecked, id)
		}
	}
	return res, nil
}

// RefreshByIDs re-reads the given matches from the provider by fixture id
// (batches of 20, events included) and applies them. Matches without a
// provider id are skipped. Used by the dev seed to fill the ticker of
// recent matches. Returns the number of requests and of changed matches.
func RefreshByIDs(ctx context.Context, app core.App, key string, matches []*core.Record) (requests, updated int, err error) {
	byAF := map[int]*core.Record{}
	var ids []int
	for _, m := range matches {
		if id := m.GetInt("afId"); id != 0 {
			byAF[id] = m
			ids = append(ids, id)
		}
	}
	structures := tournaments.NewStructureCache(app)
	client := football.New(key, 0, 0)
	for len(ids) > 0 {
		n := min(len(ids), football.MaxIDs)
		batch := ids[:n]
		ids = ids[n:]
		fx, ferr := client.ByIDs(ctx, batch)
		requests++
		if ferr != nil {
			return requests, updated, ferr
		}
		for _, f := range fx {
			rec := byAF[f.ID]
			if rec == nil {
				continue
			}
			st := structures.For(rec.GetString("tournament"))
			if st == nil {
				continue
			}
			if applyFixture(app, st, rec, f) {
				updated++
			}
		}
	}
	return requests, updated, nil
}

// Ticker backfill after a full refresh: the outage fallback. The live pass
// is the only thing that brings events in; if the server was down for a
// whole match, the full refresh still sets its result, but the ticker
// stays empty. So after each full refresh, the finished matches of the
// live-synced seasons that kicked off since the previous refresh and have
// no events are fetched by id (20 per request). Normally that's none.
const (
	// eventsLookback is how far back the first backfill after a start
	// looks (no previous refresh is known then), and the cap on the window.
	eventsLookback    = 24 * time.Hour
	eventsLookbackMax = 48 * time.Hour
	// eventsMargin overlaps the windows, so a match at the edge isn't missed.
	eventsMargin = time.Hour
)

var lastFull = map[string]time.Time{} // tournament id → previous full refresh

// BackfillEvents fetches the ticker of recently finished matches that have
// none (see above). Returns the number of requests and matches asked for.
func BackfillEvents(ctx context.Context, app core.App) (requests, asked int) {
	liveMu.Lock()
	defer liveMu.Unlock()
	key := os.Getenv("API_FOOTBALL_KEY")
	if key == "" {
		return 0, 0
	}
	now := clock.Now(app)
	var missing []*core.Record
	for _, s := range apiSeasons(app) {
		tid := s.t.Id
		since := lastFull[tid]
		if since.IsZero() || now.Sub(since) > eventsLookbackMax {
			since = now.Add(-eventsLookback)
		}
		since = since.Add(-eventsMargin)
		var ms []*core.Record
		if err := app.RecordQuery("matches").
			AndWhere(dbx.HashExp{"tournament": tid, "status": "finished"}).
			AndWhere(dbx.NewExp("afId != 0 AND kickoff >= {:since} AND kickoff <= {:now}", dbx.Params{
				"since": since.UTC().Format(types.DefaultDateLayout),
				"now":   now.UTC().Format(types.DefaultDateLayout),
			})).
			AndWhere(dbx.NewExp("NOT EXISTS (SELECT 1 FROM match_events e WHERE e.match = matches.id)")).
			All(&ms); err != nil {
			log.Printf("[sync] events backfill %s: %v", s.t.GetString("slug"), err)
			continue
		}
		lastFull[tid] = now
		missing = append(missing, ms...)
	}
	// One batch across all seasons (20 matches per request).
	if len(missing) > 0 {
		r, _, err := RefreshByIDs(ctx, app, key, missing)
		requests, asked = r, len(missing)
		if err != nil {
			log.Printf("[sync] events backfill: %v", err)
		}
	}
	if asked > 0 {
		log.Printf("[sync] events backfill: %d matches without a ticker, %d requests", asked, requests)
	}
	return requests, asked
}
