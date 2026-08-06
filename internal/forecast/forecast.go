// Package forecast backs the one-time pre-tournament prediction: full group
// standings, the manually-chosen extra-qualifier slots (WC2026: 8 best
// thirds), and the knockout bracket winners. One forecast per
// (user, tournament); editable until that tournament starts and validated
// server-side against the tournament's structure.
package forecast

import (
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/bracket"
	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/tournaments"
)

// startOf returns a tournament's Forecast deadline: its startsAt when set,
// else the earliest kickoff of its matches.
func startOf(app core.App, t *core.Record) (time.Time, error) {
	if ts := t.GetDateTime("startsAt").Time(); !ts.IsZero() {
		return ts, nil
	}
	ms, err := app.FindRecordsByFilter("matches",
		"tournament = {:t}", "kickoff", 1, 0, map[string]any{"t": t.Id})
	if err != nil || len(ms) == 0 {
		return time.Time{}, err
	}
	return ms[0].GetDateTime("kickoff").Time(), nil
}

func locked(app core.App, t *core.Record) bool {
	ts, err := startOf(app, t)
	if err != nil || ts.IsZero() {
		return false
	}
	return !clock.Now(app).Before(ts)
}

// groupTeams returns letter -> set(teamId) from a tournament's groups.
func groupTeams(app core.App, tournamentID string) (map[string]map[string]bool, error) {
	gs, err := app.FindRecordsByFilter("tournament_groups",
		"tournament = {:t}", "letter", 0, 0, map[string]any{"t": tournamentID})
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]bool{}
	for _, g := range gs {
		set := map[string]bool{}
		for _, id := range g.GetStringSlice("teams") {
			set[id] = true
		}
		out[g.GetString("letter")] = set
	}
	return out, nil
}

// validate enforces the lock and that group orderings only contain that
// group's own teams without duplicates. Partial forecasts are allowed (the
// user fills it in over multiple sessions); only clearly invalid data is
// rejected. A forecast without a tournament is assigned the current one, so
// pre-rework clients keep working.
// bypass lets the dev bot generator insert a complete Forecast regardless of
// the lock. Never set in production (dev-only path).
var bypass atomic.Bool

// SetBypass toggles the dev-only validation bypass.
func SetBypass(b bool) { bypass.Store(b) }

func validate(app core.App, rec *core.Record) error {
	if bypass.Load() {
		return nil
	}
	t, err := tournamentOf(app, rec)
	if err != nil {
		return apis.NewBadRequestError("no tournament to forecast", nil)
	}
	if locked(app, t) {
		return apis.NewBadRequestError("the tournament has started — the Forecast is locked", nil)
	}
	groups, err := groupTeams(app, t.Id)
	if err != nil {
		return err
	}
	var order map[string][]string
	if err := rec.UnmarshalJSONField("groupOrder", &order); err != nil {
		return nil // empty/!set yet — allow
	}
	for letter, ids := range order {
		members := groups[letter]
		if members == nil {
			return apis.NewBadRequestError("unknown group "+letter, nil)
		}
		seen := map[string]bool{}
		for _, id := range ids {
			if id == "" {
				continue
			}
			if !members[id] {
				return apis.NewBadRequestError("a team in group "+letter+" is not in that group", nil)
			}
			if seen[id] {
				return apis.NewBadRequestError("duplicate team in group "+letter, nil)
			}
			seen[id] = true
		}
	}
	return nil
}

// tournamentOf resolves a forecast record's tournament, defaulting (and
// stamping) the current one when unset.
func tournamentOf(app core.App, rec *core.Record) (*core.Record, error) {
	if tid := rec.GetString("tournament"); tid != "" {
		return app.FindRecordById("tournaments", tid)
	}
	t, err := tournaments.Current(app)
	if err != nil {
		return nil, err
	}
	rec.Set("tournament", t.Id)
	return t, nil
}

// ThirdSlot is a knockout match with an extra-qualifier placeholder side.
type ThirdSlot struct {
	MatchNum int      `json:"matchNum"`
	Winner   string   `json:"winner"`  // group-winner letter this slot pairs with
	Allowed  []string `json:"allowed"` // group letters eligible (fallback only)
}

// sharesLeague reports whether users a and b are both members of at least
// one common League.
func sharesLeague(app core.App, a, b string) bool {
	mine, err := app.FindRecordsByFilter("league_members",
		"user = {:u}", "", 0, 0, map[string]any{"u": a})
	if err != nil {
		return false
	}
	for _, m := range mine {
		if _, err := app.FindFirstRecordByFilter("league_members",
			"league = {:l} && user = {:u}",
			map[string]any{"l": m.GetString("league"), "u": b}); err == nil {
			return true
		}
	}
	return false
}

// resolveTournamentParam returns the tournament addressed by an optional
// ?tournament=<slug> query param, defaulting to the current one.
func resolveTournamentParam(app core.App, e *core.RequestEvent) (*core.Record, error) {
	if slug := e.Request.URL.Query().Get("tournament"); slug != "" {
		return tournaments.BySlug(app, slug)
	}
	return tournaments.Current(app)
}

// structurePayload builds everything the Forecast builder (and the bots)
// need for one tournament: its stages/shape, groups with teams, the knockout
// skeleton with placeholder labels, the extra-qualifier slots, the official
// allocation table, and the lock state.
func structurePayload(app core.App, t *core.Record) (map[string]any, error) {
	st, err := tournaments.StructureOf(t)
	if err != nil {
		return nil, err
	}
	eq := st.ExtraQualifiers

	groups, err := app.FindRecordsByFilter("tournament_groups",
		"tournament = {:t}", "letter", 0, 0, map[string]any{"t": t.Id})
	if err != nil {
		return nil, err
	}
	gOut := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		gOut = append(gOut, map[string]any{
			"letter": g.GetString("letter"),
			"teams":  g.GetStringSlice("teams"),
		})
	}

	all, err := app.FindRecordsByFilter("matches",
		"tournament = {:t}", "num", 0, 0, map[string]any{"t": t.Id})
	if err != nil {
		return nil, err
	}
	slotPrefix := ""
	if eq != nil {
		slotPrefix = strconv.Itoa(eq.FromPosition)
	}
	kOut := make([]map[string]any, 0, len(all))
	thirds := make([]ThirdSlot, 0, 8)
	for _, mt := range all {
		if !st.IsKnockout(mt.GetString("stage")) {
			continue
		}
		home := mt.GetString("homeLabel")
		away := mt.GetString("awayLabel")
		num := mt.GetInt("num")
		kOut = append(kOut, map[string]any{
			"num":       num,
			"stage":     mt.GetString("stage"),
			"round":     mt.GetString("roundLabel"),
			"homeLabel": home,
			"awayLabel": away,
		})
		if eq == nil {
			continue
		}
		for _, lbl := range []string{home, away} {
			if strings.HasPrefix(lbl, slotPrefix) && strings.Contains(lbl, "/") {
				w, _ := bracket.WinnerLetter(home, away)
				thirds = append(thirds, ThirdSlot{
					MatchNum: num,
					Winner:   w,
					Allowed:  strings.Split(strings.TrimPrefix(lbl, slotPrefix), "/"),
				})
			}
		}
	}

	var thirdTable map[string]map[string]string
	if eq != nil {
		thirdTable = bracket.Table(eq.TableKey)
	}
	ts, _ := startOf(app, t)
	return map[string]any{
		"tournament": map[string]any{
			"id":        t.Id,
			"slug":      t.GetString("slug"),
			"name":      t.GetString("name"),
			"shortName": t.GetString("shortName"),
			"status":    t.GetString("status"),
		},
		"structure":       st,
		"groups":          gOut,
		"knockout":        kOut,
		"thirdSlots":      thirds,
		"thirdTable":      thirdTable,
		"tournamentStart": ts,
		"locked":          locked(app, t),
	}, nil
}

// Register wires the Forecast validation hooks and the structure endpoints.
func Register(app core.App, se *core.ServeEvent) {
	app.OnRecordCreate("forecasts").BindFunc(func(e *core.RecordEvent) error {
		if err := validate(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdate("forecasts").BindFunc(func(e *core.RecordEvent) error {
		if err := validate(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	// GET /api/forecast/of/{userId}?tournament=<slug> — a friend's Forecast
	// for a tournament (default: current). Visible to anyone who shares a
	// League with them (no lock gate: in a friends group you want to see
	// picks right away). Not registered on the forecasts table, which stays
	// own-only.
	se.Router.GET("/api/forecast/of/{userId}", func(e *core.RequestEvent) error {
		uid := e.Request.PathValue("userId")
		if uid != e.Auth.Id && !sharesLeague(app, e.Auth.Id, uid) {
			return apis.NewForbiddenError("not in a league with this player", nil)
		}
		u, err := app.FindRecordById("users", uid)
		if err != nil {
			return apis.NewNotFoundError("user not found", nil)
		}
		t, err := resolveTournamentParam(app, e)
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		out := map[string]any{"userId": uid, "name": u.GetString("name"), "tournament": t.GetString("slug")}
		fc, err := app.FindFirstRecordByFilter("forecasts",
			"user = {:u} && tournament = {:t}", map[string]any{"u": uid, "t": t.Id})
		if err != nil {
			out["forecast"] = nil
			return e.JSON(http.StatusOK, out)
		}
		var order, bracket map[string]any
		var thirds map[string]any
		var rationale map[string]any
		_ = fc.UnmarshalJSONField("groupOrder", &order)
		_ = fc.UnmarshalJSONField("thirdQualifiers", &thirds)
		_ = fc.UnmarshalJSONField("bracket", &bracket)
		_ = fc.UnmarshalJSONField("rationale", &rationale)
		out["forecast"] = map[string]any{
			"groupOrder":      order,
			"thirdQualifiers": thirds,
			"bracket":         bracket,
			"rationale":       rationale,
		}
		return e.JSON(http.StatusOK, out)
	}).Bind(apis.RequireAuth())

	// GET /api/tournaments/{slug}/structure — the tournament-scoped
	// structure endpoint.
	se.Router.GET("/api/tournaments/{slug}/structure", func(e *core.RequestEvent) error {
		t, err := tournaments.BySlug(app, e.Request.PathValue("slug"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		out, err := structurePayload(app, t)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, out)
	}).Bind(apis.RequireAuth())

	// GET /api/forecast/structure — compat alias for the current tournament.
	se.Router.GET("/api/forecast/structure", func(e *core.RequestEvent) error {
		t, err := resolveTournamentParam(app, e)
		if err != nil {
			return apis.NewNotFoundError("no tournament", nil)
		}
		out, err := structurePayload(app, t)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, out)
	}).Bind(apis.RequireAuth())
}
