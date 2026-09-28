package h2h

import (
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/pools"
	"github.com/floholz/matchowl/internal/tournaments"
)

// registerMe wires GET /api/h2h/me[?tournament=<season id>]: the caller's
// head-to-head pools at a glance — the headline round and the next one
// from the caller's side, the to-do for each, and every pick the caller
// may see, by match id. One request feeds Home, the match lists and the
// competition hub instead of one per row (the tip drawer keeps
// /api/h2h/match/{id} for the full per-pool detail).
func registerMe(app core.App, se *core.ServeEvent) {
	se.Router.GET("/api/h2h/me", func(e *core.RequestEvent) error {
		season := e.Request.URL.Query().Get("tournament")
		mems, _ := app.FindRecordsByFilter("pool_members", "user = {:u}", "", 0, 0,
			map[string]any{"u": e.Auth.Id})
		pp := &people{app: app, cache: map[string]*person{}}
		out := make([]map[string]any, 0)
		for _, mem := range mems {
			lg, err := app.FindRecordById("pools", mem.GetString("pool"))
			if err != nil || lg.GetString("mode") != pools.ModeH2H || pools.Status(app, lg) == pools.PoolFinished {
				continue
			}
			if season != "" && pools.Season(lg) != season {
				continue
			}
			out = append(out, meView(app, lg, e.Auth.Id, pp))
		}
		return e.JSON(http.StatusOK, map[string]any{"pools": out})
	}).Bind(apis.RequireAuth())
}

// pickMark is what a match row shows: the caller's own picks and what the
// rival has revealed.
type pickMark struct {
	Saved       bool `json:"saved"`
	Banned      bool `json:"banned"`
	RivalSaved  bool `json:"rivalSaved"`
	RivalBanned bool `json:"rivalBanned"`
}

func meView(app core.App, lg *core.Record, uid string, pp *people) map[string]any {
	_ = TickPool(app, lg)
	now := clock.Now(app)
	rows, _ := roundRows(app, lg.Id)
	allow := lg.GetInt("saveCalls")
	if allow < 1 {
		allow = 1
	}
	seasonSlug, compKey := "", ""
	if t, err := app.FindRecordById("tournaments", pools.Season(lg)); err == nil {
		seasonSlug = t.GetString("slug")
		if c, err := app.FindRecordById("competitions", t.GetString("competition")); err == nil {
			compKey = c.GetString("key")
		}
	}

	// Every pick the caller may see, by match: their own across all
	// rounds, and the rival's revealed ones per opened round.
	marks := map[string]*pickMark{}
	mark := func(id string) *pickMark {
		m := marks[id]
		if m == nil {
			m = &pickMark{}
			marks[id] = m
		}
		return m
	}
	minePicks, _ := app.FindRecordsByFilter("h2h_picks", "pool = {:p} && user = {:u}", "", 0, 0,
		map[string]any{"p": lg.Id, "u": uid})
	for _, r := range minePicks {
		switch r.GetString("kind") {
		case KindSave:
			mark(r.GetString("match")).Saved = true
		case KindBan:
			mark(r.GetString("match")).Banned = true
		}
	}
	for _, row := range rows {
		rival, in := RivalIn(row, uid)
		if !in || rival == Ghost {
			continue
		}
		closed := row.GetString("status") == "closed"
		rp := revealed(app, picksFor(app, lg.Id, row.GetString("key")), closed)[rival]
		if rp == nil {
			continue
		}
		for _, m := range rp.Saves {
			mark(m).RivalSaved = true
		}
		if rp.Ban != "" {
			mark(rp.Ban).RivalBanned = true
		}
	}

	// The headline round: the earliest open one, else the last closed.
	var current *core.Record
	for _, row := range rows {
		current = row
		if row.GetString("status") == "open" {
			break
		}
	}
	// The round that opens next.
	have := map[string]bool{}
	for _, r := range rows {
		have[r.GetString("key")] = true
	}
	var next *tournaments.Round
	for _, r := range playable(app, lg, pools.Season(lg)) {
		if have[r.Key] || !r.First.After(now) {
			continue
		}
		rr := r
		next = &rr
		break
	}

	v := map[string]any{
		"poolId":      lg.Id,
		"name":        lg.GetString("name"),
		"season":      pools.Season(lg),
		"seasonSlug":  seasonSlug,
		"competition": compKey,
		"saveCalls":   allow,
		"picks":       marks,
		"current":     nil,
		"next":        nil,
	}
	if current != nil {
		v["current"] = duelState(app, lg, uid, current, nil, len(rows), allow, pp)
	}
	if next != nil {
		v["next"] = duelState(app, lg, uid, nil, next, len(rows), allow, pp)
	}
	return v
}

// duelState is one round from the caller's side: who they face, how the
// duel stands (open or closed rounds), and what is still to do (rounds
// that still take tips and picks). Either an opened round row or an
// upcoming playable round.
func duelState(app core.App, lg *core.Record, uid string, row *core.Record, up *tournaments.Round, ordinal, allow int, pp *people) map[string]any {
	now := clock.Now(app)
	var key, label string
	var num int
	var ids []string
	status := "upcoming"
	v := map[string]any{}
	if row != nil {
		key, label, num = row.GetString("key"), row.GetString("label"), row.GetInt("num")
		status = row.GetString("status")
		_ = row.UnmarshalJSONField("matches", &ids)
		v["firstKickoff"] = iso(row.GetDateTime("firstKickoff").Time())
		v["closesAt"] = iso(row.GetDateTime("closesAt").Time())
		res := ResultsOf(row)
		if res == nil {
			if r, err := Compute(app, lg, row); err == nil {
				res = r
			} else {
				res = &Results{}
			}
		}
		v["counted"] = len(res.Counted)
		rival, in := RivalIn(row, uid)
		v["paired"], v["ghost"], v["rival"] = in, in && rival == Ghost, pp.get(rival)
		pairs := make([]map[string]any, 0, len(res.Pairs))
		for _, p := range res.Pairs {
			switch uid {
			case p.A:
				v["mine"], v["theirs"], v["pts"] = p.ScoreA, p.ScoreB, p.PtsA
			case p.B:
				v["mine"], v["theirs"], v["pts"] = p.ScoreB, p.ScoreA, p.PtsB
			}
			pairs = append(pairs, map[string]any{
				"a": pp.get(p.A), "b": pp.get(p.B),
				"scoreA": p.ScoreA, "scoreB": p.ScoreB, "ptsA": p.PtsA, "ptsB": p.PtsB,
			})
		}
		v["pairs"] = pairs
	} else {
		key, label, num, ids = up.Key, up.Label, up.Num, up.MatchIDs
		v["firstKickoff"] = iso(up.First)
		v["closesAt"] = iso(up.Last.Add(CloseGrace))
		v["counted"] = 0
		rival, in := "", false
		for _, p := range Pairings(roster(app, lg.Id), ordinal) {
			if p.A == uid {
				rival, in = p.B, true
			} else if p.B == uid {
				rival, in = p.A, true
			}
		}
		v["paired"], v["ghost"], v["rival"] = in, in && rival == Ghost, pp.get(rival)
	}
	v["key"], v["label"], v["num"], v["name"], v["status"], v["matches"] = key, label, num, RoundName(label, num), status, len(ids)

	if status != "closed" {
		untipped := 0
		for _, id := range ids {
			m, err := app.FindRecordById("matches", id)
			if err != nil || !now.Before(m.GetDateTime("kickoff").Time()) {
				continue
			}
			if _, err := app.FindFirstRecordByFilter("tips", "user = {:u} && match = {:m}",
				map[string]any{"u": uid, "m": id}); err != nil {
				untipped++
			}
		}
		mine := picksFor(app, lg.Id, key)[uid]
		saves, ban := 0, false
		if mine != nil {
			saves, ban = len(mine.Saves), mine.Ban != ""
		}
		v["untipped"], v["saveCallsLeft"], v["banPlaced"] = untipped, allow-saves, ban
	}
	return v
}
