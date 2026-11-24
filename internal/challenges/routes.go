package challenges

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/friends"
	"github.com/floholz/matchowl/internal/users"
)

func bad(e *core.RequestEvent, code int, msg string) error {
	return e.JSON(code, map[string]string{"error": msg})
}

// Routes wires /api/challenges:
//
//	GET  /api/challenges                 incoming, open and recent challenges
//	GET  /api/challenges/records         my record against each friend
//	GET  /api/challenges/match/{id}      my challenges on one match
//	POST /api/challenges                 { match, opponent, stake } send one
//	POST /api/challenges/{id}/accept     the opponent accepts (before kick-off)
//	POST /api/challenges/{id}/decline    the opponent declines
//	POST /api/challenges/{id}/cancel     the challenger withdraws a pending one
func Routes(app core.App, se *core.ServeEvent) {
	g := se.Router.Group("/api/challenges")
	g.Bind(apis.RequireAuth())
	// Friends only, so verified accounts only (as the friends graph).
	g.Bind(users.RequireVerified())

	g.GET("", func(e *core.RequestEvent) error {
		me := e.Auth.Id
		now := clock.Now(app)
		rows, err := app.FindRecordsByFilter(Collection,
			"challenger = {:u} || opponent = {:u}", "-created", 200, 0, map[string]any{"u": me})
		if err != nil {
			return err
		}
		incoming, open, recent := []map[string]any{}, []map[string]any{}, []map[string]any{}
		for _, c := range rows {
			v := view(app, c, me, now)
			if v == nil {
				continue
			}
			st := v["status"].(string)
			switch {
			case st == Pending && c.GetString("opponent") == me:
				incoming = append(incoming, v)
			case st == Pending || (st == Accepted && c.GetString("settledAt") == ""):
				open = append(open, v)
			case st == Accepted:
				if len(recent) < 10 {
					recent = append(recent, v)
				}
			}
		}
		sortViews(incoming, false)
		sortViews(open, false)
		sortViews(recent, true)
		return e.JSON(http.StatusOK, map[string]any{"incoming": incoming, "open": open, "recent": recent})
	})

	g.GET("/records", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]any{"records": Records(app, e.Auth.Id)})
	})

	g.GET("/match/{id}", func(e *core.RequestEvent) error {
		me := e.Auth.Id
		rows, err := app.FindRecordsByFilter(Collection,
			"match = {:m} && (challenger = {:u} || opponent = {:u})", "-created", 0, 0,
			map[string]any{"m": e.Request.PathValue("id"), "u": me})
		if err != nil {
			return err
		}
		now := clock.Now(app)
		out := []map[string]any{}
		for _, c := range rows {
			if v := view(app, c, me, now); v != nil && v["status"] != Declined && v["status"] != Cancelled {
				out = append(out, v)
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"challenges": out})
	})

	g.POST("", func(e *core.RequestEvent) error {
		var body struct {
			Match    string `json:"match"`
			Opponent string `json:"opponent"`
			Stake    string `json:"stake"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		me := e.Auth.Id
		if body.Opponent == "" || body.Opponent == me {
			return bad(e, http.StatusBadRequest, "pick a friend to challenge")
		}
		isFriend := false
		for _, id := range friends.AcceptedIDs(app, me) {
			if id == body.Opponent {
				isFriend = true
				break
			}
		}
		if !isFriend {
			return bad(e, http.StatusForbidden, "you can only challenge your friends")
		}
		m, err := app.FindRecordById("matches", body.Match)
		if err != nil {
			return bad(e, http.StatusNotFound, "no such match")
		}
		if started(m, clock.Now(app)) {
			return bad(e, http.StatusConflict, "this match has kicked off")
		}
		if m.GetString("homeTeam") == "" || m.GetString("awayTeam") == "" {
			return bad(e, http.StatusConflict, "the teams of this match aren't known yet")
		}
		if _, err := app.FindFirstRecordByFilter(Collection,
			"match = {:m} && (status = 'pending' || status = 'accepted') && "+
				"((challenger = {:a} && opponent = {:b}) || (challenger = {:b} && opponent = {:a}))",
			map[string]any{"m": m.Id, "a": me, "b": body.Opponent}); err == nil {
			return bad(e, http.StatusConflict, "you already have a challenge with them on this match")
		}
		col, err := app.FindCollectionByNameOrId(Collection)
		if err != nil {
			return err
		}
		c := core.NewRecord(col)
		c.Set("match", m.Id)
		c.Set("challenger", me)
		c.Set("opponent", body.Opponent)
		c.Set("stake", CleanStake(body.Stake))
		c.Set("status", Pending)
		if err := app.Save(c); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, view(app, c, me, clock.Now(app)))
	})

	// answer moves a challenge on; who may do it and from which status.
	answer := func(by, from, to string) func(e *core.RequestEvent) error {
		return func(e *core.RequestEvent) error {
			c, err := app.FindRecordById(Collection, e.Request.PathValue("id"))
			if err != nil || c.GetString(by) != e.Auth.Id {
				return bad(e, http.StatusNotFound, "no such challenge")
			}
			m, err := app.FindRecordById("matches", c.GetString("match"))
			if err != nil {
				return bad(e, http.StatusNotFound, "no such match")
			}
			now := clock.Now(app)
			if c.GetString("status") != from {
				return bad(e, http.StatusConflict, "this challenge is "+Status(c, m, now)+" already")
			}
			if started(m, now) {
				return bad(e, http.StatusConflict, "this match has kicked off")
			}
			c.Set("status", to)
			if err := app.Save(c); err != nil {
				return err
			}
			return e.JSON(http.StatusOK, view(app, c, e.Auth.Id, now))
		}
	}
	g.POST("/{id}/accept", answer("opponent", Pending, Accepted))
	g.POST("/{id}/decline", answer("opponent", Pending, Declined))
	g.POST("/{id}/cancel", answer("challenger", Pending, Cancelled))
}

func person(app core.App, id string) map[string]any {
	u, err := app.FindRecordById("users", id)
	if err != nil {
		return map[string]any{"userId": id, "name": "a former member", "avatar": "", "avatarPreset": ""}
	}
	return map[string]any{"userId": u.Id, "name": u.GetString("name"), "avatar": u.GetString("avatar"), "avatarPreset": u.GetString("avatarPreset")}
}

func teamView(app core.App, id, label string) map[string]any {
	v := map[string]any{"teamId": id, "name": label, "fifaCode": "", "iso2": "", "logo": ""}
	if t, err := app.FindRecordById("teams", id); err == nil {
		v["name"] = t.GetString("name")
		v["fifaCode"] = t.GetString("fifaCode")
		v["iso2"] = t.GetString("iso2")
		v["logo"] = t.GetString("logo")
	}
	return v
}

func tipOf(app core.App, uid, mid string) map[string]any {
	t, err := app.FindFirstRecordByFilter("tips", "user = {:u} && match = {:m}", map[string]any{"u": uid, "m": mid})
	if err != nil {
		return nil
	}
	return map[string]any{
		"ftHome": t.GetInt("ftHome"), "ftAway": t.GetInt("ftAway"),
		"etHome": t.GetInt("etHome"), "etAway": t.GetInt("etAway"),
		"penWinner": t.GetString("penWinner"),
	}
}

// view is a challenge from my side. The other side's tip shows only once
// the match has kicked off; before that, only whether they've tipped.
func view(app core.App, c *core.Record, me string, now time.Time) map[string]any {
	m, err := app.FindRecordById("matches", c.GetString("match"))
	if err != nil {
		return nil
	}
	other, mine, theirs := sides(c, me)
	mv := map[string]any{
		"id": m.Id, "kickoff": m.GetString("kickoff"), "status": m.GetString("status"),
		"finalizedAt": m.GetString("finalizedAt"),
		"ftHome":      m.GetInt("ftHome"), "ftAway": m.GetInt("ftAway"),
		"home": teamView(app, m.GetString("homeTeam"), m.GetString("homeLabel")),
		"away": teamView(app, m.GetString("awayTeam"), m.GetString("awayLabel")),
	}
	if t, err := app.FindRecordById("tournaments", m.GetString("tournament")); err == nil {
		mv["tournament"] = map[string]any{"id": t.Id, "name": t.GetString("name"), "shortName": t.GetString("shortName")}
	}
	theirTip := tipOf(app, other, m.Id)
	v := map[string]any{
		"id":          c.Id,
		"status":      Status(c, m, now),
		"stake":       c.GetString("stake"),
		"created":     c.GetString("created"),
		"mine":        c.GetString("challenger") == me, // I sent it
		"them":        person(app, other),
		"match":       mv,
		"myTip":       tipOf(app, me, m.Id),
		"theirTipped": theirTip != nil,
		"settled":     c.GetString("settledAt") != "",
	}
	if started(m, now) {
		v["theirTip"] = theirTip
	}
	if c.GetString("settledAt") != "" {
		v["myPts"] = mine
		v["theirPts"] = theirs
		v["outcome"] = Outcome(mine, theirs)
	}
	return v
}
