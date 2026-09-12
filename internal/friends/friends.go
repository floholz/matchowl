// Package friends is the mutual social graph (PLAN.md §7): one side
// requests, the other accepts. Friends power the Friends board — me and my
// friends ranked for one tournament — and nothing else; pools (internal/
// leagues) are the invite-code groups with chat and owner tools.
package friends

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/scoring"
	"github.com/floholz/matchowl/internal/tournaments"
	"github.com/floholz/matchowl/internal/users"
)

const collection = "friendships"

func bad(e *core.RequestEvent, code int, msg string) error {
	return e.JSON(code, map[string]string{"error": msg})
}

type person struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

func personOf(app core.App, id string) (person, bool) {
	u, err := app.FindRecordById("users", id)
	if err != nil {
		return person{}, false
	}
	return person{UserID: u.Id, Name: u.GetString("name"), Avatar: u.GetString("avatar")}, true
}

// edge finds the friendship row between two users in either direction.
func edge(app core.App, a, b string) *core.Record {
	rec, err := app.FindFirstRecordByFilter(collection,
		"(requester = {:a} && receiver = {:b}) || (requester = {:b} && receiver = {:a})",
		map[string]any{"a": a, "b": b})
	if err != nil {
		return nil
	}
	return rec
}

// AcceptedIDs returns the ids of the user's accepted friends.
func AcceptedIDs(app core.App, userID string) []string {
	recs, err := app.FindRecordsByFilter(collection,
		"status = 'accepted' && (requester = {:u} || receiver = {:u})", "", 0, 0,
		map[string]any{"u": userID})
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		other := r.GetString("requester")
		if other == userID {
			other = r.GetString("receiver")
		}
		out = append(out, other)
	}
	return out
}

// Register wires /api/friends.
func Register(app core.App, se *core.ServeEvent) {
	g := se.Router.Group("/api/friends")
	g.Bind(apis.RequireAuth())

	// GET /api/friends — accepted friends, incoming requests, outgoing requests.
	g.GET("", func(e *core.RequestEvent) error {
		me := e.Auth.Id
		recs, err := app.FindRecordsByFilter(collection,
			"requester = {:u} || receiver = {:u}", "-updated", 0, 0, map[string]any{"u": me})
		if err != nil {
			return err
		}
		friends, incoming, outgoing := []person{}, []person{}, []person{}
		for _, r := range recs {
			req, rec := r.GetString("requester"), r.GetString("receiver")
			other := req
			if other == me {
				other = rec
			}
			p, ok := personOf(app, other)
			if !ok {
				continue
			}
			switch {
			case r.GetString("status") == "accepted":
				friends = append(friends, p)
			case rec == me:
				incoming = append(incoming, p)
			default:
				outgoing = append(outgoing, p)
			}
		}
		return e.JSON(http.StatusOK, map[string]any{
			"friends": friends, "incoming": incoming, "outgoing": outgoing,
		})
	})

	// GET /api/friends/search?q=name — people to befriend (not bots, not you).
	g.GET("/search", func(e *core.RequestEvent) error {
		q := strings.TrimSpace(e.Request.URL.Query().Get("q"))
		if len(q) < 2 {
			return e.JSON(http.StatusOK, map[string]any{"users": []person{}})
		}
		// Starts-with on the name or on any word of it (case-insensitive),
		// not a loose contains: "bo" finds "Bob" and "Anna Bode", not "Jacob".
		recs, err := app.FindRecordsByFilter("users",
			"(name ~ {:p} || name ~ {:w}) && id != {:me} && role != 'bot'", "name", 10, 0,
			map[string]any{"p": q + "%", "w": "% " + q + "%", "me": e.Auth.Id})
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, u := range recs {
			state := ""
			if f := edge(app, e.Auth.Id, u.Id); f != nil {
				state = f.GetString("status")
				if state == "pending" && f.GetString("receiver") == e.Auth.Id {
					state = "incoming"
				}
			}
			out = append(out, map[string]any{
				"userId": u.Id, "name": u.GetString("name"), "avatar": u.GetString("avatar"), "state": state,
			})
		}
		return e.JSON(http.StatusOK, map[string]any{"users": out})
	})

	type body struct {
		UserID string `json:"userId"`
	}
	target := func(e *core.RequestEvent) (string, error) {
		var b body
		if err := e.BindBody(&b); err != nil {
			return "", err
		}
		if b.UserID == "" || b.UserID == e.Auth.Id {
			return "", bad(e, http.StatusBadRequest, "pick another user")
		}
		u, err := app.FindRecordById("users", b.UserID)
		if err != nil || users.IsBot(u) {
			return "", bad(e, http.StatusNotFound, "no such user")
		}
		return b.UserID, nil
	}

	// POST /api/friends/request { userId } — request; accepts a pending
	// request from the other side instead of crossing it.
	g.POST("/request", func(e *core.RequestEvent) error {
		other, err := target(e)
		if err != nil {
			return err
		}
		if f := edge(app, e.Auth.Id, other); f != nil {
			if f.GetString("status") == "pending" && f.GetString("receiver") == e.Auth.Id {
				f.Set("status", "accepted")
				if err := app.Save(f); err != nil {
					return err
				}
				return e.JSON(http.StatusOK, map[string]any{"state": "accepted"})
			}
			return e.JSON(http.StatusOK, map[string]any{"state": f.GetString("status")})
		}
		col, err := app.FindCollectionByNameOrId(collection)
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		rec.Set("requester", e.Auth.Id)
		rec.Set("receiver", other)
		rec.Set("status", "pending")
		if err := app.Save(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"state": "pending"})
	})

	// POST /api/friends/accept { userId } — accept their request.
	g.POST("/accept", func(e *core.RequestEvent) error {
		other, err := target(e)
		if err != nil {
			return err
		}
		f := edge(app, e.Auth.Id, other)
		if f == nil || f.GetString("status") != "pending" || f.GetString("receiver") != e.Auth.Id {
			return bad(e, http.StatusNotFound, "no request to accept")
		}
		f.Set("status", "accepted")
		if err := app.Save(f); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"state": "accepted"})
	})

	// POST /api/friends/remove { userId } — decline, withdraw or unfriend.
	g.POST("/remove", func(e *core.RequestEvent) error {
		other, err := target(e)
		if err != nil {
			return err
		}
		if f := edge(app, e.Auth.Id, other); f != nil {
			if err := app.Delete(f); err != nil {
				return err
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"state": ""})
	})

	// GET /api/friends/board?tournament=<slug> — me and my friends, ranked
	// for one tournament (default: the current one), default scoring.
	g.GET("/board", func(e *core.RequestEvent) error {
		var trec *core.Record
		var err error
		if slug := e.Request.URL.Query().Get("tournament"); slug != "" {
			trec, err = tournaments.BySlug(app, slug)
		} else {
			trec, err = tournaments.Current(app)
		}
		if err != nil {
			return bad(e, http.StatusNotFound, "no such tournament")
		}
		ids := append([]string{e.Auth.Id}, AcceptedIDs(app, e.Auth.Id)...)
		rows := scoring.Board(app, ids, "", []string{trec.Id})
		return e.JSON(http.StatusOK, map[string]any{
			"tournament": trec.GetString("slug"),
			"rows":       rows,
		})
	})
}
