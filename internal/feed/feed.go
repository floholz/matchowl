// Package feed backs the app's center surface (PLAN-feed.md): one
// chronological stream of matches across every tournament the user plays,
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

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/forecast"
	"github.com/floholz/matchowl/internal/players"
	"github.com/floholz/matchowl/internal/tournaments"
)

// window caps: how far the feed reaches by default and at most.
const (
	defaultPastDays   = 2
	defaultAheadDays  = 7
	maxWindowDays     = 30
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

// feedMatch pairs a match with its (played) tournament id.
type feedMatch struct {
	m   *core.Record
	tid string
}

// Register wires GET /api/feed.
func Register(app core.App, se *core.ServeEvent) {
	se.Router.GET("/api/feed", func(e *core.RequestEvent) error {
		uid := e.Auth.Id
		playedIDs, err := players.PlayedIDs(app, uid)
		if err != nil {
			return err
		}

		now := clock.Now(app)
		past := intParam(e, "past", defaultPastDays, maxWindowDays)
		ahead := intParam(e, "days", defaultAheadDays, maxWindowDays)
		from := now.AddDate(0, 0, -past).Truncate(24 * time.Hour)
		to := now.AddDate(0, 0, ahead+1).Truncate(24 * time.Hour)

		type tInfo struct {
			rec       *core.Record
			structure *tournaments.Structure
			view      map[string]any
		}
		infos := map[string]*tInfo{}
		playing := make([]map[string]any, 0, len(playedIDs))
		for _, tid := range playedIDs {
			t, err := app.FindRecordById("tournaments", tid)
			if err != nil {
				continue
			}
			st, err := tournaments.StructureOf(t)
			if err != nil {
				continue
			}
			v := map[string]any{
				"id":        t.Id,
				"slug":      t.GetString("slug"),
				"name":      t.GetString("name"),
				"shortName": t.GetString("shortName"),
				"status":    t.GetString("status"),
			}
			infos[tid] = &tInfo{rec: t, structure: st, view: v}
			playing = append(playing, v)
		}

		// Matches of played tournaments inside the window, kickoff-sorted.
		var window []feedMatch
		for tid := range infos {
			ms, err := app.FindRecordsByFilter("matches",
				"tournament = {:t} && kickoff >= {:from} && kickoff < {:to}",
				"kickoff", 0, 0,
				map[string]any{"t": tid, "from": from.UTC().Format(types.DefaultDateLayout), "to": to.UTC().Format(types.DefaultDateLayout)})
			if err != nil {
				continue
			}
			for _, m := range ms {
				window = append(window, feedMatch{m: m, tid: tid})
			}
		}
		sortFeed(window)
		if len(window) > maxMatchesPerLoad {
			window = window[:maxMatchesPerLoad]
		}

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
		for _, fm := range window {
			for _, f := range []string{"homeTeam", "awayTeam", "penWinner", "advancer"} {
				if id := fm.m.GetString(f); id != "" {
					teamIDs[id] = true
				}
			}
		}
		teams := map[string]map[string]any{}
		for id := range teamIDs {
			if t, err := app.FindRecordById("teams", id); err == nil {
				teams[id] = map[string]any{
					"name":     t.GetString("name"),
					"fifaCode": t.GetString("fifaCode"),
					"iso2":     t.GetString("iso2"),
					"clubKey":  t.GetString("clubKey"),
				}
			}
		}

		out := make([]map[string]any, 0, len(window))
		for _, fm := range window {
			m, info := fm.m, infos[fm.tid]
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
			}
			if tip := tipByMatch[m.Id]; tip != nil {
				myTip := map[string]any{
					"ftHome": tip.GetInt("ftHome"), "ftAway": tip.GetInt("ftAway"),
					"etHome": tip.GetInt("etHome"), "etAway": tip.GetInt("etAway"),
					"penWinner": tip.GetString("penWinner"), "advancer": tip.GetString("advancer"),
				}
				if pts, ok := pointsByMatch[m.Id]; ok && row["finished"] == true {
					myTip["points"] = pts
				}
				row["myTip"] = myTip
			}
			out = append(out, row)
		}

		// Deadline cards: played tournaments whose forecast is still open.
		deadlines := make([]map[string]any, 0)
		for tid, info := range infos {
			status := info.rec.GetString("status")
			if status != tournaments.StatusUpcoming && status != tournaments.StatusActive {
				continue
			}
			lock, err := forecast.StartOf(app, info.rec)
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
				"tournament":  info.view,
				"locksAt":     lock.UTC().Format(time.RFC3339),
				"hasForecast": has,
			})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"now":       now.UTC().Format(time.RFC3339),
			"playing":   playing,
			"matches":   out,
			"teams":     teams,
			"deadlines": deadlines,
		})
	}).Bind(apis.RequireAuth())
}

func stageNameOf(st *tournaments.Structure, code string) string {
	if s := st.Stage(code); s != nil {
		return s.Name
	}
	return code
}

func sortFeed(window []feedMatch) {
	sort.SliceStable(window, func(i, j int) bool {
		a, b := window[i].m, window[j].m
		if ka, kb := a.GetString("kickoff"), b.GetString("kickoff"); ka != kb {
			return ka < kb
		}
		return a.GetInt("num") < b.GetInt("num")
	})
}
