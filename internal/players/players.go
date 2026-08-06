// Package players owns tournament participation (PLAN-feed.md): who plays
// which tournament. Playing is what puts a tournament's matches into your
// feed. Subscription is hybrid: an explicit Play button, plus
// auto-subscribe the moment you submit your first tip or forecast in a
// tournament. League activity never auto-joins — it only powers the
// suggestions endpoint.
package players

import (
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/tournaments"
)

const collection = "tournament_players"

// Ensure records the user as a player of the tournament (idempotent).
// source is "manual" (Play button) or "auto" (first tip/forecast).
func Ensure(app core.App, tournamentID, userID, source string) error {
	if tournamentID == "" || userID == "" {
		return nil
	}
	if _, err := app.FindFirstRecordByFilter(collection,
		"tournament = {:t} && user = {:u}",
		map[string]any{"t": tournamentID, "u": userID}); err == nil {
		return nil
	}
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("tournament", tournamentID)
	rec.Set("user", userID)
	rec.Set("source", source)
	return app.Save(rec)
}

// PlayedIDs returns the ids of tournaments the user plays.
func PlayedIDs(app core.App, userID string) ([]string, error) {
	recs, err := app.FindRecordsByFilter(collection,
		"user = {:u}", "-joinedAt", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.GetString("tournament"))
	}
	return out, nil
}

// leagueMates returns the set of user ids sharing at least one private
// league with the user (the auto-managed Global league is excluded — it
// contains everyone).
func leagueMates(app core.App, userID string) (map[string]bool, error) {
	mates := map[string]bool{}
	mine, err := app.FindRecordsByFilter("league_members",
		"user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}
	globalID := ""
	if g, err := app.FindFirstRecordByFilter("leagues",
		"inviteCode = {:c}", map[string]any{"c": "GLOBAL"}); err == nil {
		globalID = g.Id
	}
	for _, m := range mine {
		lid := m.GetString("league")
		if lid == globalID {
			continue
		}
		members, err := app.FindRecordsByFilter("league_members",
			"league = {:l}", "", 0, 0, map[string]any{"l": lid})
		if err != nil {
			continue
		}
		for _, mm := range members {
			if uid := mm.GetString("user"); uid != userID {
				mates[uid] = true
			}
		}
	}
	return mates, nil
}

// Register wires the participation endpoints and the auto-subscribe hooks.
func Register(app core.App, se *core.ServeEvent) {
	// Auto-subscribe: your first tip or forecast in a tournament makes you
	// a player. AfterCreateSuccess so validation has already passed.
	app.OnRecordAfterCreateSuccess("tips").BindFunc(func(e *core.RecordEvent) error {
		if match, err := e.App.FindRecordById("matches", e.Record.GetString("match")); err == nil {
			if err := Ensure(e.App, match.GetString("tournament"), e.Record.GetString("user"), "auto"); err != nil {
				log.Printf("[players] auto-subscribe (tip): %v", err)
			}
		}
		return e.Next()
	})
	app.OnRecordAfterCreateSuccess("forecasts").BindFunc(func(e *core.RecordEvent) error {
		if err := Ensure(e.App, e.Record.GetString("tournament"), e.Record.GetString("user"), "auto"); err != nil {
			log.Printf("[players] auto-subscribe (forecast): %v", err)
		}
		return e.Next()
	})

	// POST /api/tournaments/{slug}/play — the explicit Play button.
	se.Router.POST("/api/tournaments/{slug}/play", func(e *core.RequestEvent) error {
		t, err := tournaments.BySlug(app, e.Request.PathValue("slug"))
		if err != nil || t.GetString("status") == tournaments.StatusDraft {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		if err := Ensure(app, t.Id, e.Auth.Id, "manual"); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true, "tournament": t.GetString("slug")})
	}).Bind(apis.RequireAuth())

	// DELETE /api/tournaments/{slug}/play — leave. Tips/forecasts are kept
	// (leaving only removes the tournament from your feed).
	se.Router.DELETE("/api/tournaments/{slug}/play", func(e *core.RequestEvent) error {
		t, err := tournaments.BySlug(app, e.Request.PathValue("slug"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		if rec, err := app.FindFirstRecordByFilter(collection,
			"tournament = {:t} && user = {:u}",
			map[string]any{"t": t.Id, "u": e.Auth.Id}); err == nil {
			if err := app.Delete(rec); err != nil {
				return err
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	}).Bind(apis.RequireAuth())

	// GET /api/me/tournaments — the tournaments the caller plays.
	se.Router.GET("/api/me/tournaments", func(e *core.RequestEvent) error {
		ids, err := PlayedIDs(app, e.Auth.Id)
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			if t, err := app.FindRecordById("tournaments", id); err == nil {
				out = append(out, map[string]any{
					"id":     t.Id,
					"slug":   t.GetString("slug"),
					"name":   t.GetString("name"),
					"status": t.GetString("status"),
				})
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"tournaments": out})
	}).Bind(apis.RequireAuth())

	// GET /api/tournaments/suggestions — non-played, non-archived
	// tournaments, each with how many private-league mates play it. The
	// client decides how to surface them (feed cards).
	se.Router.GET("/api/tournaments/suggestions", func(e *core.RequestEvent) error {
		playedList, err := PlayedIDs(app, e.Auth.Id)
		if err != nil {
			return err
		}
		played := map[string]bool{}
		for _, id := range playedList {
			played[id] = true
		}
		mates, err := leagueMates(app, e.Auth.Id)
		if err != nil {
			return err
		}
		recs, err := tournaments.All(app)
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0)
		for _, t := range recs {
			st := t.GetString("status")
			if played[t.Id] || st == tournaments.StatusDraft || st == tournaments.StatusArchived {
				continue
			}
			n := 0
			if len(mates) > 0 {
				rows, _ := app.FindRecordsByFilter(collection,
					"tournament = {:t}", "", 0, 0, map[string]any{"t": t.Id})
				for _, r := range rows {
					if mates[r.GetString("user")] {
						n++
					}
				}
			}
			out = append(out, map[string]any{
				"id":          t.Id,
				"slug":        t.GetString("slug"),
				"name":        t.GetString("name"),
				"status":      st,
				"startsAt":    t.GetString("startsAt"),
				"leagueMates": n,
			})
		}
		return e.JSON(http.StatusOK, map[string]any{"suggestions": out})
	}).Bind(apis.RequireAuth())
}
