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
	closed := row.GetString("status") == "closed"
	shown := revealed(app, res.Picks, closed)
	pairs := make([]map[string]any, 0, len(res.Pairs))
	for _, p := range res.Pairs {
		var savesA, savesB int
		if up := shown[p.A]; up != nil {
			savesA = len(up.Saves)
		}
		if up := shown[p.B]; up != nil {
			savesB = len(up.Saves)
		}
		pairs = append(pairs, map[string]any{
			"a": pp.get(p.A), "b": pp.get(p.B),
			"scoreA": p.ScoreA, "scoreB": p.ScoreB, "ptsA": p.PtsA, "ptsB": p.PtsB,
			// Revealed picks: save calls that have kicked off, and whether
			// the rival's ban is out.
			"savesA": savesA, "savesB": savesB,
			"bannedA": shown[p.B] != nil && shown[p.B].Ban != "",
			"bannedB": p.B != Ghost && shown[p.A] != nil && shown[p.A].Ban != "",
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
		v["picks"] = shown
	}
	return v
}

// picksView is the picks panel for one member and round: the round's
// matches with the member's own save calls and ban, what the rival has
// revealed, and the allowance left.
func picksView(app core.App, lg *core.Record, uid string, r tournaments.Round, pp *people) map[string]any {
	now := clock.Now(app)
	all := picksFor(app, lg.Id, r.Key)
	mine := all[uid]
	if mine == nil {
		mine = &UserPicks{Saves: []string{}}
	}
	rival, in := rivalOf(app, lg, r.Key, uid)
	var rivalPicks *UserPicks
	if in && rival != Ghost {
		rivalPicks = revealed(app, all, false)[rival]
	}
	closed := false
	status := "upcoming"
	if row, _ := app.FindFirstRecordByFilter("h2h_rounds", "pool = {:p} && key = {:k}",
		map[string]any{"p": lg.Id, "k": r.Key}); row != nil {
		status = row.GetString("status")
		closed = status == "closed"
	}
	matches := make([]map[string]any, 0, len(r.MatchIDs))
	for _, id := range r.MatchIDs {
		m, err := app.FindRecordById("matches", id)
		if err != nil {
			continue
		}
		v := matchView(app, m, now)
		v["saved"] = mine.saved(id)
		v["banned"] = mine.Ban == id
		v["rivalSaved"] = rivalPicks != nil && rivalPicks.saved(id)
		v["rivalBanned"] = rivalPicks != nil && rivalPicks.Ban == id
		matches = append(matches, v)
	}
	allow := lg.GetInt("saveCalls")
	if allow < 1 {
		allow = 1
	}
	out := map[string]any{
		"round": map[string]any{
			"key": r.Key, "label": r.Label, "num": r.Num, "status": status,
			"firstKickoff": iso(r.First), "closesAt": iso(r.Last.Add(CloseGrace)),
		},
		"saveCalls":     allow,
		"saveCallsLeft": allow - len(mine.Saves),
		"mine":          mine,
		"paired":        in,
		"rival":         pp.get(rival), // nil = the Ghost (or not paired)
		"ghost":         in && rival == Ghost,
		"closed":        closed,
		"matches":       matches,
	}
	return out
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

	// GET /api/pools/{id}/h2h/picks?round=<key> — my save calls and ban for
	// a round (default: the headline round, else the next one), with what
	// the rival has revealed.
	g.GET("/picks", func(e *core.RequestEvent) error {
		lg, err := member(e)
		if err != nil {
			return err
		}
		key := e.Request.URL.Query().Get("round")
		var round *tournaments.Round
		for _, r := range playable(app, lg, pools.Season(lg)) {
			if key == "" {
				// Default: the earliest round that has not closed.
				row, _ := app.FindFirstRecordByFilter("h2h_rounds", "pool = {:p} && key = {:k}",
					map[string]any{"p": lg.Id, "k": r.Key})
				if row == nil || row.GetString("status") != "closed" {
					round = &r
					break
				}
			} else if r.Key == key {
				round = &r
				break
			}
		}
		if round == nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "no such matchday"})
		}
		pp := &people{app: app, cache: map[string]*person{}}
		return e.JSON(http.StatusOK, picksView(app, lg, e.Auth.Id, *round, pp))
	})

	// POST /api/pools/{id}/h2h/picks { "match", "kind": "save"|"ban", "on": true|false }
	g.POST("/picks", func(e *core.RequestEvent) error {
		lg, err := member(e)
		if err != nil {
			return err
		}
		var body struct {
			Match string `json:"match"`
			Kind  string `json:"kind"`
			On    *bool  `json:"on"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		on := body.On == nil || *body.On
		if err := SetPick(app, lg, e.Auth.Id, body.Match, body.Kind, on); err != nil {
			if pe, ok := err.(*PickError); ok {
				return e.JSON(pe.Status, map[string]string{"error": pe.Msg})
			}
			return err
		}
		m, _ := app.FindRecordById("matches", body.Match)
		var round *tournaments.Round
		if m != nil {
			round, _ = roundOf(app, lg, m)
		}
		if round == nil {
			return e.JSON(http.StatusOK, map[string]any{"ok": true})
		}
		pp := &people{app: app, cache: map[string]*person{}}
		return e.JSON(http.StatusOK, picksView(app, lg, e.Auth.Id, *round, pp))
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
