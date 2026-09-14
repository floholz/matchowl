package h2h

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/pools"
	"github.com/floholz/matchowl/internal/tournaments"
)

// person is the member slice the client renders; nil = the Ghost.
type person struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Role   string `json:"role"`
}

type people struct {
	app   core.App
	cache map[string]*person
}

func (p *people) get(uid string) *person {
	if uid == Ghost {
		return nil
	}
	if v, ok := p.cache[uid]; ok {
		return v
	}
	u, err := p.app.FindRecordById("users", uid)
	v := &person{UserID: uid, Name: "Former member"}
	if err == nil {
		v.Name, v.Avatar, v.Role = u.GetString("name"), u.GetString("avatar"), u.GetString("role")
	}
	p.cache[uid] = v
	return v
}

func iso(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// roundView renders one round. Open rounds carry provisional results
// computed now; closed rounds their frozen ones. withBreakdown adds the
// per-member, per-match points (the round page).
func roundView(app core.App, lg *core.Record, row *core.Record, pp *people, withBreakdown bool) map[string]any {
	res := ResultsOf(row)
	if res == nil {
		if r, err := Compute(app, lg, row); err == nil {
			res = r
		} else {
			res = &Results{}
		}
	}
	pairs := make([]map[string]any, 0, len(res.Pairs))
	for _, p := range res.Pairs {
		pairs = append(pairs, map[string]any{
			"a": pp.get(p.A), "b": pp.get(p.B),
			"scoreA": p.ScoreA, "scoreB": p.ScoreB, "ptsA": p.PtsA, "ptsB": p.PtsB,
		})
	}
	var ids []string
	_ = row.UnmarshalJSONField("matches", &ids)
	v := map[string]any{
		"key":          row.GetString("key"),
		"label":        row.GetString("label"),
		"num":          row.GetInt("num"),
		"ordinal":      row.GetInt("ordinal"),
		"status":       row.GetString("status"),
		"firstKickoff": iso(row.GetDateTime("firstKickoff").Time()),
		"closesAt":     iso(row.GetDateTime("closesAt").Time()),
		"pairs":        pairs,
		"matches":      len(ids),
		"counted":      len(res.Counted),
		"ghost":        res.Ghost,
	}
	if withBreakdown {
		v["matchIds"] = ids
		v["countedIds"] = res.Counted
		v["breakdown"] = res.Breakdown
	}
	return v
}

// nextView previews the round that opens next: which matchday, when, and
// the pairings the current roster would get — the "who do I play next".
func nextView(app core.App, lg *core.Record, rows []*core.Record, pp *people) map[string]any {
	now := clock.Now(app)
	have := map[string]bool{}
	for _, r := range rows {
		have[r.GetString("key")] = true
	}
	for _, r := range playable(app, lg, pools.Season(lg)) {
		if have[r.Key] || !r.First.After(now) {
			continue
		}
		pairs := make([]map[string]any, 0)
		for _, p := range Pairings(roster(app, lg.Id), len(rows)) {
			pairs = append(pairs, map[string]any{"a": pp.get(p.A), "b": pp.get(p.B)})
		}
		return map[string]any{
			"key": r.Key, "label": r.Label, "num": r.Num,
			"firstKickoff": iso(r.First), "closesAt": iso(r.Last.Add(CloseGrace)),
			"matches": len(r.MatchIDs), "pairs": pairs,
		}
	}
	return nil
}

// Register wires the head-to-head routes and the job.
func Register(app core.App, se *core.ServeEvent) {
	app.Cron().MustAdd("h2h-tick", "*/5 * * * *", func() { Tick(app) })

	member := func(e *core.RequestEvent) (*core.Record, error) {
		id := e.Request.PathValue("id")
		if _, err := app.FindFirstRecordByFilter("pool_members", "pool = {:l} && user = {:u}",
			map[string]any{"l": id, "u": e.Auth.Id}); err != nil {
			return nil, e.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this pool"})
		}
		lg, err := app.FindRecordById("pools", id)
		if err != nil {
			return nil, e.JSON(http.StatusNotFound, map[string]string{"error": "pool not found"})
		}
		if lg.GetString("mode") != pools.ModeH2H {
			return nil, e.JSON(http.StatusBadRequest, map[string]string{"error": "this pool is not head-to-head"})
		}
		return lg, nil
	}

	g := se.Router.Group("/api/pools/{id}/h2h")
	g.Bind(apis.RequireAuth())

	// GET /api/pools/{id}/h2h — the table, every round (open ones with
	// provisional scores), the round to show first, and the next pairing.
	g.GET("", func(e *core.RequestEvent) error {
		lg, err := member(e)
		if err != nil {
			return err
		}
		_ = TickPool(app, lg)
		rows, err := roundRows(app, lg.Id)
		if err != nil {
			return err
		}
		pp := &people{app: app, cache: map[string]*person{}}
		// The round to headline: the earliest open one (a round can open
		// early when one of its matches was pulled forward), else the last
		// closed one.
		views := make([]map[string]any, 0, len(rows))
		current, firstOpen := "", ""
		for _, row := range rows {
			views = append(views, roundView(app, lg, row, pp, false))
			current = row.GetString("key")
			if firstOpen == "" && row.GetString("status") == "open" {
				firstOpen = current
			}
		}
		if firstOpen != "" {
			current = firstOpen
		}
		var first tournaments.Round
		if pr := playable(app, lg, pools.Season(lg)); len(pr) > 0 {
			first = pr[0]
		}
		return e.JSON(http.StatusOK, map[string]any{
			"table":      Table(app, lg, rows),
			"rounds":     views,
			"current":    current,
			"next":       nextView(app, lg, rows, pp),
			"firstRound": map[string]any{"key": first.Key, "label": first.Label, "firstKickoff": iso(first.First)},
			"saveCalls":  lg.GetInt("saveCalls"),
		})
	})

	// GET /api/pools/{id}/h2h/round?key=<stage|label> — one round with the
	// per-match breakdown behind every score.
	g.GET("/round", func(e *core.RequestEvent) error {
		lg, err := member(e)
		if err != nil {
			return err
		}
		row, err := app.FindFirstRecordByFilter("h2h_rounds", "pool = {:p} && key = {:k}",
			map[string]any{"p": lg.Id, "k": e.Request.URL.Query().Get("key")})
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "no such round"})
		}
		pp := &people{app: app, cache: map[string]*person{}}
		return e.JSON(http.StatusOK, roundView(app, lg, row, pp, true))
	})
}
