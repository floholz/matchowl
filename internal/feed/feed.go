// Package feed backs the app's center surface (docs/plans/04-feed-nav-forecast.md): one
// chronological stream of "my matches" (see package scope), or of every
// season or one competition,
// plus the deadline moments (forecast locks) woven in. The server returns a
// flat kickoff-sorted window with everything denormalized (tournament,
// teams, my tip, my points); the client groups by the user's local day —
// grouping server-side by UTC date would mislabel evening kickoffs for most
// of Europe.
package feed

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/forecast"
	"github.com/floholz/matchowl/internal/scope"
	wmsync "github.com/floholz/matchowl/internal/sync"
	"github.com/floholz/matchowl/internal/teams"
	"github.com/floholz/matchowl/internal/tournaments"
)

// window caps: how far the feed reaches by default and at most.
const (
	defaultPastDays   = 2
	defaultAheadDays  = 7
	maxWindowDays     = 30
	maxTeamWindowDays = 400
	maxMatchesPerLoad = 400 // hard backstop across all played tournaments
)

func intParam(e *core.RequestEvent, name string, def, max int) int {
	v := def
	if s := e.Request.URL.Query().Get(name); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			v = n
		}
	}
	if v > max {
		v = max
	}
	return v
}

// Register wires GET /api/feed.
func Register(app core.App, se *core.ServeEvent) {
	se.Router.GET("/api/feed", func(e *core.RequestEvent) error {
		uid := e.Auth.Id
		mine, err := scope.Mine(app, uid)
		if err != nil {
			return err
		}

		now := clock.Now(app)
		q := e.Request.URL.Query()
		teamKey := q.Get("team")
		maxDays := maxWindowDays
		if teamKey != "" {
			maxDays = maxTeamWindowDays // a team plays a few matches a month
		}
		past := intParam(e, "past", defaultPastDays, maxDays)
		ahead := intParam(e, "days", defaultAheadDays, maxDays)
		from := now.AddDate(0, 0, -past).Truncate(24 * time.Hour)
		to := now.AddDate(0, 0, ahead+1).Truncate(24 * time.Hour)

		// Which matches: ?scope=mine (default) is "my matches"; ?scope=all
		// (or the older ?all=1) every visible season; ?scope=country the
		// matches of a team from the user's country; ?competition=<key>
		// every open season of one competition; ?team=<key> every match of
		// one team.
		which := q.Get("scope")
		if q.Get("all") == "1" {
			which = "all"
		}
		compKey := q.Get("competition")
		var window []*core.Record
		switch {
		case compKey != "" || which == "all":
			ids, err := visibleSeasons(app, compKey)
			if err != nil {
				return err
			}
			if len(ids) > 0 {
				if err := app.RecordQuery("matches").
					AndWhere(dbx.In("tournament", ids...)).
					AndWhere(dbx.NewExp("kickoff >= {:from} AND kickoff < {:to}", dbx.Params{
						"from": from.UTC().Format(types.DefaultDateLayout),
						"to":   to.UTC().Format(types.DefaultDateLayout),
					})).
					OrderBy("kickoff ASC", "num ASC").
					All(&window); err != nil {
					return err
				}
			}
		case teamKey != "" || which == "country":
			var idents []string
			if teamKey != "" {
				if id, err := app.FindFirstRecordByFilter(teams.Collection, "key = {:k}", map[string]any{"k": teamKey}); err == nil {
					idents = append(idents, id.Id)
				}
			} else if c := e.Auth.GetString("country"); c != "" {
				recs, err := app.FindRecordsByFilter(teams.Collection, "country = {:c}", "", 0, 0, map[string]any{"c": c})
				if err != nil {
					return err
				}
				for _, r := range recs {
					idents = append(idents, r.Id)
				}
			}
			rows := teams.SeasonRowIDs(app, idents)
			ids, err := visibleSeasons(app, "")
			if err != nil {
				return err
			}
			if len(rows) > 0 && len(ids) > 0 {
				in := make([]any, len(rows))
				for i, r := range rows {
					in[i] = r
				}
				if err := app.RecordQuery("matches").
					AndWhere(dbx.In("tournament", ids...)).
					AndWhere(dbx.Or(dbx.In("homeTeam", in...), dbx.In("awayTeam", in...))).
					AndWhere(dbx.NewExp("kickoff >= {:from} AND kickoff < {:to}", dbx.Params{
						"from": from.UTC().Format(types.DefaultDateLayout),
						"to":   to.UTC().Format(types.DefaultDateLayout),
					})).
					OrderBy("kickoff ASC", "num ASC").
					All(&window); err != nil {
					return err
				}
			}
		default:
			if window, err = mine.Matches(app, from, to); err != nil {
				return err
			}
		}
		if len(window) > maxMatchesPerLoad {
			window = window[:maxMatchesPerLoad]
		}

		type tInfo struct {
			rec       *core.Record
			structure *tournaments.Structure
			view      map[string]any
		}
		infos := map[string]*tInfo{}
		info := func(tid string) *tInfo {
			if in, ok := infos[tid]; ok {
				return in
			}
			infos[tid] = nil
			t, err := app.FindRecordById("tournaments", tid)
			if err != nil {
				return nil
			}
			st, err := tournaments.StructureOf(t)
			if err != nil {
				return nil
			}
			v := map[string]any{
				"id":        t.Id,
				"slug":      t.GetString("slug"),
				"name":      t.GetString("name"),
				"shortName": t.GetString("shortName"),
				"status":    t.GetString("status"),
			}
			// Competition key so rows can link to /competitions/{key}.
			if c, err := app.FindRecordById("competitions", t.GetString("competition")); err == nil {
				v["competition"] = c.GetString("key")
			}
			infos[tid] = &tInfo{rec: t, structure: st, view: v}
			return infos[tid]
		}
		// The seasons your pools bind, for the competition chips.
		poolSeasons := make([]map[string]any, 0, len(mine.Seasons))
		for tid := range mine.Seasons {
			if in := info(tid); in != nil {
				poolSeasons = append(poolSeasons, in.view)
			}
		}
		sort.Slice(poolSeasons, func(i, j int) bool {
			return poolSeasons[i]["name"].(string) < poolSeasons[j]["name"].(string)
		})
		kept := window[:0]
		for _, m := range window {
			if info(m.GetString("tournament")) != nil {
				kept = append(kept, m)
			}
		}
		window = kept

		// My tips + my default-config points, mapped by match id.
		tipByMatch := map[string]*core.Record{}
		if tps, err := app.FindRecordsByFilter("tips",
			"user = {:u}", "", 0, 0, map[string]any{"u": uid}); err == nil {
			for _, t := range tps {
				tipByMatch[t.GetString("match")] = t
			}
		}
		pointsByMatch := map[string]int{}
		if def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true"); err == nil {
			if rows, err := app.FindRecordsByFilter("match_scores",
				"user = {:u} && config = {:c}", "", 0, 0,
				map[string]any{"u": uid, "c": def.Id}); err == nil {
				for _, r := range rows {
					pointsByMatch[r.GetString("match")] = r.GetInt("points")
				}
			}
		}

		// Teams referenced by the window, denormalized once.
		teamIDs := map[string]bool{}
		for _, m := range window {
			for _, f := range []string{"homeTeam", "awayTeam", "penWinner", "advancer"} {
				if id := m.GetString(f); id != "" {
					teamIDs[id] = true
				}
			}
		}
		teamViews := map[string]map[string]any{}
		identKeys := map[string]string{}
		for id := range teamIDs {
			if t, err := app.FindRecordById("teams", id); err == nil {
				v := map[string]any{
					"name":     t.GetString("name"),
					"fifaCode": t.GetString("fifaCode"),
					"iso2":     t.GetString("iso2"),
					"clubKey":  t.GetString("clubKey"),
					"logo":     t.GetString("logo"),
				}
				// The team page key (/teams/{key}).
				if iid := t.GetString("identity"); iid != "" {
					k, ok := identKeys[iid]
					if !ok {
						if ir, err := app.FindRecordById(teams.Collection, iid); err == nil {
							k = ir.GetString("key")
						}
						identKeys[iid] = k
					}
					v["key"] = k
				}
				teamViews[id] = v
			}
		}

		out := make([]map[string]any, 0, len(window))
		for _, m := range window {
			info := infos[m.GetString("tournament")]
			stage := m.GetString("stage")
			// Rows carry the same field names as the matches collection so
			// the client can feed them straight into the shared match card.
			row := map[string]any{
				"id":          m.Id,
				"tournament":  info.view,
				"stage":       stage,
				"stageName":   stageNameOf(info.structure, stage),
				"knockout":    info.structure.IsKnockout(stage),
				"groupLetter": m.GetString("groupLetter"),
				"roundLabel":  m.GetString("roundLabel"),
				"num":         m.GetInt("num"),
				"kickoff":     m.GetString("kickoff"),
				"status":      m.GetString("status"),
				"finalizedAt": m.GetString("finalizedAt"),
				"homeTeam":    m.GetString("homeTeam"),
				"awayTeam":    m.GetString("awayTeam"),
				"homeLabel":   m.GetString("homeLabel"),
				"awayLabel":   m.GetString("awayLabel"),
				"ftHome":      m.GetInt("ftHome"),
				"ftAway":      m.GetInt("ftAway"),
				"etHome":      m.GetInt("etHome"),
				"etAway":      m.GetInt("etAway"),
				"penHome":     m.GetInt("penHome"),
				"penAway":     m.GetInt("penAway"),
				"advancer":    m.GetString("advancer"),
				"livePhase":   m.GetString("livePhase"),
				"liveMinute":  m.GetInt("liveMinute"),
				"liveExtra":   m.GetInt("liveExtra"),
				"liveAt":      m.GetString("liveAt"),
				// Why the match is yours ("pool", "tip"); empty outside
				// your scope (the All and competition filters).
				"reasons": orEmpty(mine.Reasons(m)),
			}
			// Two-legged ties: the other leg is usually outside the feed
			// window, so denormalize what the card needs (result + which leg
			// this is + where to jump).
			if info.structure.IsKnockout(stage) {
				if other, first := wmsync.OtherLeg(app, info.structure, m); other != nil {
					row["leg"] = map[string]any{
						"id":          other.Id,
						"first":       first,
						"kickoff":     other.GetString("kickoff"),
						"status":      other.GetString("status"),
						"finalizedAt": other.GetString("finalizedAt"),
						"homeTeam":    other.GetString("homeTeam"),
						"awayTeam":    other.GetString("awayTeam"),
						"ftHome":      other.GetInt("ftHome"),
						"ftAway":      other.GetInt("ftAway"),
						"etHome":      other.GetInt("etHome"),
						"etAway":      other.GetInt("etAway"),
					}
				}
			}
			if tip := tipByMatch[m.Id]; tip != nil {
				myTip := map[string]any{
					"ftHome": tip.GetInt("ftHome"), "ftAway": tip.GetInt("ftAway"),
					"etHome": tip.GetInt("etHome"), "etAway": tip.GetInt("etAway"),
					"penWinner": tip.GetString("penWinner"), "advancer": tip.GetString("advancer"),
				}
				finished := m.GetString("status") == "finished" || m.GetString("finalizedAt") != ""
				if pts, ok := pointsByMatch[m.Id]; ok && finished {
					myTip["points"] = pts
				}
				row["myTip"] = myTip
			}
			out = append(out, row)
		}

		// Deadline cards: your pools' seasons whose forecast is still open.
		deadlines := make([]map[string]any, 0)
		for tid := range mine.Seasons {
			in := info(tid)
			if in == nil {
				continue
			}
			status := in.rec.GetString("status")
			if status != tournaments.StatusUpcoming && status != tournaments.StatusActive {
				continue
			}
			lock, err := forecast.StartOf(app, in.rec)
			if err != nil || lock.IsZero() || !now.Before(lock) {
				continue
			}
			has := false
			if _, err := app.FindFirstRecordByFilter("forecasts",
				"user = {:u} && tournament = {:t}",
				map[string]any{"u": uid, "t": tid}); err == nil {
				has = true
			}
			deadlines = append(deadlines, map[string]any{
				"type":        "forecast",
				"tournament":  in.view,
				"locksAt":     lock.UTC().Format(time.RFC3339),
				"hasForecast": has,
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"now":         now.UTC().Format(time.RFC3339),
			"poolSeasons": poolSeasons,
			"matches":     out,
			"teams":       teamViews,
			"deadlines":   deadlines,
		})
	}).Bind(apis.RequireAuth())
}

func stageNameOf(st *tournaments.Structure, code string) string {
	if s := st.Stage(code); s != nil {
		return s.Name
	}
	return code
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// visibleSeasons lists the ids of every non-draft season, or of one
// competition's seasons that aren't archived when a key is given.
func visibleSeasons(app core.App, compKey string) ([]any, error) {
	recs, err := tournaments.All(app)
	if err != nil {
		return nil, err
	}
	compID := ""
	if compKey != "" {
		c, err := app.FindFirstRecordByFilter("competitions", "key = {:k}", map[string]any{"k": compKey})
		if err != nil {
			return nil, nil
		}
		compID = c.Id
	}
	out := make([]any, 0, len(recs))
	for _, r := range recs {
		st := r.GetString("status")
		if st == tournaments.StatusDraft {
			continue
		}
		if compID != "" && (r.GetString("competition") != compID || st == tournaments.StatusArchived) {
			continue
		}
		out = append(out, r.Id)
	}
	return out, nil
}
