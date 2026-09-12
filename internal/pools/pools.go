// Package leagues provides the private-competition ("Pool") endpoints:
// create (with a unique invite code, creator auto-joined as owner), join by
// code, list mine, and a leaderboard. Scoring totals are filled by the Phase 5
// engine; until then the leaderboard returns members with zeroed points.
package pools

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/scoring"
	"github.com/floholz/matchowl/internal/tournaments"
)

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars

// GlobalInviteCode is the fixed invite code of the auto-managed "Global" league
// that every registered user belongs to.
const GlobalInviteCode = "GLOBAL"

func newInviteCode(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	var sb strings.Builder
	for _, v := range b {
		sb.WriteByte(codeAlphabet[int(v)%len(codeAlphabet)])
	}
	return sb.String()
}

func bad(e *core.RequestEvent, code int, msg string) error {
	return e.JSON(code, map[string]string{"error": msg})
}

// tournamentIDs resolves season slugs to ids. Only running or upcoming
// seasons can be bound — a pool for a finished season makes no sense.
func tournamentIDs(app core.App, slugs []string) ([]string, error) {
	out := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		t, err := tournaments.BySlug(app, slug)
		if err != nil {
			return nil, fmt.Errorf("unknown season %q", slug)
		}
		switch t.GetString("status") {
		case tournaments.StatusActive, tournaments.StatusUpcoming:
		default:
			return nil, fmt.Errorf("season %q is not open for pools", slug)
		}
		out = append(out, t.Id)
	}
	return out, nil
}

// Pool states, derived from the bound seasons: live while any of them
// runs, upcoming before the first kick-off, finished once all are over,
// open when nothing is bound (Global).
const (
	PoolLive     = "live"
	PoolUpcoming = "upcoming"
	PoolFinished = "finished"
	PoolOpen     = "open"
)

// Finished reports whether every bound season is over: the pool is an
// archive then — read-only except for "Set up next season".
func Finished(app core.App, lg *core.Record) bool {
	return poolStatus(app, lg) == PoolFinished
}

// ChatGrace is how long a finished pool's chat stays open after its last
// season ends, so the outcome can still be argued over.
const ChatGrace = 30 * 24 * time.Hour

// ChatUntil is when a finished pool's chat closes (zero while any season is
// still on, or when the pool is unbound).
func ChatUntil(app core.App, lg *core.Record) time.Time {
	if !Finished(app, lg) {
		return time.Time{}
	}
	var last time.Time
	for _, id := range lg.GetStringSlice("tournaments") {
		t, err := app.FindRecordById("tournaments", id)
		if err != nil {
			continue
		}
		if e := t.GetDateTime("endsAt").Time(); e.After(last) {
			last = e
		}
	}
	if last.IsZero() {
		return time.Time{}
	}
	return last.Add(ChatGrace)
}

// ChatOpen reports whether the pool's chat still takes messages.
func ChatOpen(app core.App, lg *core.Record) bool {
	until := ChatUntil(app, lg)
	return until.IsZero() || time.Now().Before(until)
}

func poolStatus(app core.App, lg *core.Record) string {
	ids := lg.GetStringSlice("tournaments")
	if len(ids) == 0 {
		return PoolOpen
	}
	live, upcoming := false, false
	for _, id := range ids {
		t, err := app.FindRecordById("tournaments", id)
		if err != nil {
			continue
		}
		switch t.GetString("status") {
		case tournaments.StatusActive:
			live = true
		case tournaments.StatusUpcoming:
			upcoming = true
		}
	}
	switch {
	case live:
		return PoolLive
	case upcoming:
		return PoolUpcoming
	default:
		return PoolFinished
	}
}

// nextSeasons maps a pool's bound seasons to the same competitions' latest
// open season (running first, else the next upcoming), skipping competitions
// without one — the default for "Set up next season".
func nextSeasons(app core.App, lg *core.Record) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range lg.GetStringSlice("tournaments") {
		t, err := app.FindRecordById("tournaments", id)
		if err != nil {
			continue
		}
		comp := t.GetString("competition")
		if comp == "" || seen[comp] {
			continue
		}
		seen[comp] = true
		var best *core.Record
		recs, _ := app.FindRecordsByFilter("tournaments",
			"competition = {:c} && (status = 'active' || status = 'upcoming')", "startsAt", 0, 0,
			map[string]any{"c": comp})
		for _, r := range recs {
			if best == nil || (r.GetString("status") == tournaments.StatusActive && best.GetString("status") != tournaments.StatusActive) {
				best = r
			}
		}
		if best != nil {
			out = append(out, best.Id)
		}
	}
	return out
}

func chatUntilView(app core.App, lg *core.Record) string {
	if u := ChatUntil(app, lg); !u.IsZero() {
		return u.UTC().Format(time.RFC3339)
	}
	return ""
}

// seasonViews lists a pool's bound seasons for the client.
func seasonViews(app core.App, lg *core.Record) []map[string]any {
	out := make([]map[string]any, 0)
	for _, id := range lg.GetStringSlice("tournaments") {
		t, err := app.FindRecordById("tournaments", id)
		if err != nil {
			continue
		}
		v := map[string]any{
			"id": t.Id, "slug": t.GetString("slug"), "name": t.GetString("name"),
			"shortName": t.GetString("shortName"), "status": t.GetString("status"),
		}
		if c, err := app.FindRecordById("competitions", t.GetString("competition")); err == nil {
			v["competition"] = map[string]any{
				"key": c.GetString("key"), "name": c.GetString("name"), "shortName": c.GetString("shortName"),
			}
		}
		out = append(out, v)
	}
	return out
}

// extraCodes are non-qualifier team codes we still allow in invite codes purely
// for fun — e.g. Italy, who didn't make WC2026. They are NOT real tournament
// teams (no group/fixtures), only flavour in the scoreline-style codes.
var extraCodes = []string{"ITA"}

// teamCodes returns the uppercase 3-letter FIFA codes used to mint
// scoreline-style invite codes: all seeded teams plus the extraCodes flavour
// set, de-duplicated.
func teamCodes(app core.App) []string {
	seen := make(map[string]bool)
	codes := make([]string, 0, 48+len(extraCodes))
	add := func(c string) {
		if c = strings.ToUpper(strings.TrimSpace(c)); c != "" && !seen[c] {
			seen[c] = true
			codes = append(codes, c)
		}
	}
	if teams, err := app.FindRecordsByFilter("teams", "id != ''", "", 0, 0); err == nil {
		for _, t := range teams {
			add(t.GetString("fifaCode"))
		}
	}
	for _, c := range extraCodes {
		add(c)
	}
	return codes
}

// newScoreCode builds a scoreline-style invite code — TEAM·digit·TEAM·digit,
// e.g. "AUT6GER2" — from two distinct team codes and two single-digit "scores".
// Returns "" when fewer than two team codes are available so the caller can
// fall back to a plain random code.
func newScoreCode(codes []string) string {
	if len(codes) < 2 {
		return ""
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	home := int(b[0]) % len(codes)
	away := int(b[1]) % len(codes)
	if away == home { // keep the two teams distinct (a match has two sides)
		away = (away + 1) % len(codes)
	}
	return fmt.Sprintf("%s%d%s%d", codes[home], int(b[2])%10, codes[away], int(b[3])%10)
}

// uniqueCode returns a fresh invite code not currently in use. It prefers a
// scoreline-style code (e.g. "AUT6GER2"); if team data is missing or the themed
// space is somehow exhausted, it falls back to a plain random 6-char code so
// league creation never breaks.
func uniqueCode(app core.App) string {
	codes := teamCodes(app)
	for range 20 {
		code := newScoreCode(codes)
		if code == "" {
			break
		}
		if _, err := app.FindFirstRecordByFilter("pools", "inviteCode = {:c}", map[string]any{"c": code}); err != nil {
			return code // not found => unique
		}
	}
	var code string
	for range 10 {
		code = newInviteCode(6)
		if _, err := app.FindFirstRecordByFilter("pools", "inviteCode = {:c}", map[string]any{"c": code}); err != nil {
			break
		}
	}
	return code
}

// ownedLeague loads the league and authorizes the caller as its owner. It
// returns a 403 for non-owners — which also covers the auto-managed "Global"
// league, whose owner is empty and therefore matches no authenticated user.
func ownedLeague(app core.App, e *core.RequestEvent, id string) (*core.Record, error) {
	lg, err := app.FindRecordById("pools", id)
	if err != nil {
		return nil, bad(e, http.StatusNotFound, "pool not found")
	}
	if lg.GetString("owner") != e.Auth.Id {
		return nil, bad(e, http.StatusForbidden, "only the pool owner can do this")
	}
	return lg, nil
}

// ownedOpenLeague is ownedLeague for changes a finished pool no longer takes.
func ownedOpenLeague(app core.App, e *core.RequestEvent, id string) (*core.Record, error) {
	lg, err := ownedLeague(app, e, id)
	if err != nil {
		return nil, err
	}
	if Finished(app, lg) {
		return nil, bad(e, http.StatusConflict, "this pool is finished — set it up again for the next season")
	}
	return lg, nil
}

// Register wires the League endpoints. Most require an authenticated user;
// the invite-preview route below is intentionally public.
func Register(app core.App, se *core.ServeEvent) {
	// Auto-managed "Global" league: ensure it exists, backfill existing users,
	// and add every new user as a member when their account is created.
	if err := backfillGlobal(app); err != nil {
		log.Printf("[pools] global backfill failed: %v", err)
	}
	app.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		if err := ensureGlobalMember(e.App, e.Record.Id); err != nil {
			log.Printf("[pools] auto-join global failed for %s: %v", e.Record.Id, err)
		}
		return e.Next()
	})

	// Public: resolve an invite code to a league name for the invite landing
	// page. Possessing the code is the capability (it's an invite link); only
	// id + name are exposed, nothing member- or score-related.
	//
	// Lives under /api/invite (not /api/leagues) on purpose: Go 1.22's router
	// rejects a path-param route under /api/leagues/ as ambiguous against
	// /api/leagues/{id}/leaderboard.
	se.Router.GET("/api/invite/{code}", func(e *core.RequestEvent) error {
		code := strings.ToUpper(strings.TrimSpace(e.Request.PathValue("code")))
		league, err := app.FindFirstRecordByFilter("pools",
			"inviteCode = {:c}", map[string]any{"c": code})
		if err != nil {
			return bad(e, http.StatusNotFound, "invalid invite code")
		}
		return e.JSON(http.StatusOK, map[string]any{
			"id": league.Id, "name": league.GetString("name"),
		})
	})

	g := se.Router.Group("/api/pools")
	g.Bind(apis.RequireAuth())

	// POST /api/leagues/create  { "name": "..." }
	g.POST("/create", func(e *core.RequestEvent) error {
		var body struct {
			Name        string   `json:"name"`
			Tournaments []string `json:"tournaments"` // season slugs the pool counts
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return bad(e, http.StatusBadRequest, "name required")
		}
		tids, err := tournamentIDs(app, body.Tournaments)
		if err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}

		col, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}

		code := uniqueCode(app)

		def, _ := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")

		league := core.NewRecord(col)
		league.Set("name", name)
		league.Set("inviteCode", code)
		league.Set("owner", e.Auth.Id)
		if def != nil {
			league.Set("scoringConfig", def.Id)
		}
		league.Set("tournaments", tids)
		if err := app.Save(league); err != nil {
			return err
		}
		if err := addMember(app, league.Id, e.Auth.Id, "owner"); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{
			"id": league.Id, "name": name, "inviteCode": code,
		})
	})

	// POST /api/leagues/join  { "code": "ABC123" }
	g.POST("/join", func(e *core.RequestEvent) error {
		var body struct {
			Code string `json:"code"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		code := strings.ToUpper(strings.TrimSpace(body.Code))
		league, err := app.FindFirstRecordByFilter("pools", "inviteCode = {:c}", map[string]any{"c": code})
		if err != nil {
			return bad(e, http.StatusNotFound, "invalid invite code")
		}
		if Finished(app, league) {
			return bad(e, http.StatusConflict, "this pool is finished")
		}
		if existing, _ := app.FindFirstRecordByFilter("pool_members",
			"pool = {:l} && user = {:u}",
			map[string]any{"l": league.Id, "u": e.Auth.Id}); existing != nil {
			return e.JSON(http.StatusOK, map[string]any{"id": league.Id, "name": league.GetString("name"), "already": true})
		}
		if err := addMember(app, league.Id, e.Auth.Id, "member"); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"id": league.Id, "name": league.GetString("name")})
	})

	// GET /api/leagues/mine
	g.GET("/mine", func(e *core.RequestEvent) error {
		members, err := app.FindRecordsByFilter("pool_members",
			"user = {:u}", "-joinedAt", 0, 0, map[string]any{"u": e.Auth.Id})
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0, len(members))
		for _, m := range members {
			lg, err := app.FindRecordById("pools", m.GetString("pool"))
			if err != nil {
				continue
			}
			cnt, _ := app.CountRecords("pool_members",
				dbx.HashExp{"pool": lg.Id})
			role := m.GetString("role")
			private := lg.GetBool("privateCode")
			// On a private league only the owner may see/share the code.
			code := lg.GetString("inviteCode")
			if private && role != "owner" {
				code = ""
			}
			out = append(out, map[string]any{
				"id":          lg.Id,
				"name":        lg.GetString("name"),
				"inviteCode":  code,
				"role":        role,
				"private":     private,
				"members":     cnt,
				"tournaments": seasonViews(app, lg),
				"status":      poolStatus(app, lg),
				"chatOpen":    ChatOpen(app, lg),
				"chatUntil":   chatUntilView(app, lg),
			})
		}
		return e.JSON(http.StatusOK, map[string]any{"pools": out})
	})

	// GET /api/leagues/{id}/leaderboard?tournament=<slug> — standings for one
	// tournament (default: current). Leagues persist across tournaments.
	g.GET("/{id}/leaderboard", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if _, err := app.FindFirstRecordByFilter("pool_members",
			"pool = {:l} && user = {:u}",
			map[string]any{"l": id, "u": e.Auth.Id}); err != nil {
			return bad(e, http.StatusForbidden, "not a member of this pool")
		}
		lg, err := app.FindRecordById("pools", id)
		if err != nil {
			return bad(e, http.StatusNotFound, "pool not found")
		}
		bound := lg.GetStringSlice("tournaments")
		// ?tournament=<slug> narrows to one season; a pool without it sums
		// its bound seasons; Global (unbound) falls back to the current one.
		var tids []string
		used := ""
		if slug := e.Request.URL.Query().Get("tournament"); slug != "" {
			trec, terr := tournaments.BySlug(app, slug)
			if terr != nil {
				return bad(e, http.StatusNotFound, "no such tournament")
			}
			tids, used = []string{trec.Id}, trec.GetString("slug")
		} else if len(bound) > 0 {
			tids = bound
		} else {
			trec, terr := tournaments.Current(app)
			if terr != nil {
				return bad(e, http.StatusNotFound, "no such tournament")
			}
			tids, used = []string{trec.Id}, trec.GetString("slug")
		}
		lb, err := scoring.Leaderboard(app, id, tids)
		if err != nil {
			return bad(e, http.StatusNotFound, "pool not found")
		}
		lb["tournament"] = used
		lb["tournaments"] = seasonViews(app, lg)
		lb["status"] = poolStatus(app, lg)
		lb["chatOpen"] = ChatOpen(app, lg)
		lb["chatUntil"] = chatUntilView(app, lg)
		// Include the league's scoring config so the legend can render it
		// without the client reading the (now members-only) leagues table.
		if lg, err := app.FindRecordById("pools", id); err == nil {
			cid := lg.GetString("scoringConfig")
			var sc *core.Record
			if cid != "" {
				sc, _ = app.FindRecordById("scoring_configs", cid)
			}
			if sc == nil {
				sc, _ = app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
			}
			if sc != nil {
				var cfg map[string]any
				if json.Unmarshal([]byte(sc.GetString("config")), &cfg) == nil {
					lb["scoring"] = cfg
				}
			}
		}
		return e.JSON(http.StatusOK, lb)
	})

	// ---- Owner-only management (rename, regenerate code, privacy, remove) ----

	// POST /api/leagues/{id}/tournaments { "tournaments": ["slug", …] } —
	// the seasons the pool counts (owner only).
	g.POST("/{id}/tournaments", func(e *core.RequestEvent) error {
		lg, err := ownedOpenLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			Tournaments []string `json:"tournaments"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		tids, err := tournamentIDs(app, body.Tournaments)
		if err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		lg.Set("tournaments", tids)
		if err := app.Save(lg); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"tournaments": seasonViews(app, lg)})
	})

	// POST /api/leagues/{id}/clone { "name": "...", "tournaments": [...] } —
	// "Set up next season": a new pool with the same members (the caller
	// as owner), settings and a fresh invite code; the old one stays.
	g.POST("/{id}/clone", func(e *core.RequestEvent) error {
		src, err := ownedLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			Name        string   `json:"name"`
			Tournaments []string `json:"tournaments"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			name = src.GetString("name")
		}
		var tids []string
		if len(body.Tournaments) > 0 {
			tids, err = tournamentIDs(app, body.Tournaments)
			if err != nil {
				return bad(e, http.StatusBadRequest, err.Error())
			}
		} else {
			tids = nextSeasons(app, src)
		}
		if len(tids) == 0 {
			return bad(e, http.StatusBadRequest, "none of this pool's competitions has an open season yet")
		}
		col, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}
		next := core.NewRecord(col)
		next.Set("name", name)
		next.Set("inviteCode", uniqueCode(app))
		next.Set("owner", e.Auth.Id)
		next.Set("scoringConfig", src.GetString("scoringConfig"))
		next.Set("privateCode", src.GetBool("privateCode"))
		next.Set("tournaments", tids)
		if err := app.Save(next); err != nil {
			return err
		}
		members, _ := app.FindRecordsByFilter("pool_members",
			"pool = {:l}", "", 0, 0, map[string]any{"l": src.Id})
		for _, m := range members {
			role := "member"
			if m.GetString("user") == e.Auth.Id {
				role = "owner"
			}
			if err := addMember(app, next.Id, m.GetString("user"), role); err != nil {
				return err
			}
		}
		return e.JSON(http.StatusOK, map[string]any{
			"id": next.Id, "name": name, "inviteCode": next.GetString("inviteCode"),
		})
	})

	// POST /api/leagues/{id}/rename  { "name": "..." }
	g.POST("/{id}/rename", func(e *core.RequestEvent) error {
		lg, err := ownedOpenLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return bad(e, http.StatusBadRequest, "name required")
		}
		lg.Set("name", name)
		if err := app.Save(lg); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"id": lg.Id, "name": name})
	})

	// POST /api/leagues/{id}/code/regenerate
	g.POST("/{id}/code/regenerate", func(e *core.RequestEvent) error {
		lg, err := ownedOpenLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		code := uniqueCode(app)
		lg.Set("inviteCode", code)
		if err := app.Save(lg); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"inviteCode": code})
	})

	// POST /api/leagues/{id}/code/visibility  { "private": true }
	g.POST("/{id}/code/visibility", func(e *core.RequestEvent) error {
		lg, err := ownedOpenLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			Private bool `json:"private"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		lg.Set("privateCode", body.Private)
		if err := app.Save(lg); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"private": body.Private})
	})

	// POST /api/leagues/{id}/members/remove  { "userId": "..." }
	g.POST("/{id}/members/remove", func(e *core.RequestEvent) error {
		lg, err := ownedOpenLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			UserID string `json:"userId"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		if body.UserID == "" {
			return bad(e, http.StatusBadRequest, "userId required")
		}
		if body.UserID == lg.GetString("owner") {
			return bad(e, http.StatusBadRequest, "the owner cannot be removed")
		}
		member, err := app.FindFirstRecordByFilter("pool_members",
			"pool = {:l} && user = {:u}",
			map[string]any{"l": lg.Id, "u": body.UserID})
		if err != nil {
			return bad(e, http.StatusNotFound, "not a member of this pool")
		}
		if err := app.Delete(member); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})

	// GET /api/leagues/{id}/bots — bot accounts NOT yet in this league, so the
	// owner can add them. Owner-only (the management panel is owner-only).
	g.GET("/{id}/bots", func(e *core.RequestEvent) error {
		lg, err := ownedLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		bots, err := app.FindRecordsByFilter("users", "role = 'bot'", "name", 0, 0)
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0, len(bots))
		for _, b := range bots {
			if existing, _ := app.FindFirstRecordByFilter("pool_members",
				"pool = {:l} && user = {:u}",
				map[string]any{"l": lg.Id, "u": b.Id}); existing != nil {
				continue // already a member
			}
			out = append(out, map[string]any{
				"userId":  b.Id,
				"name":    b.GetString("name"),
				"avatar":  b.GetString("avatar"),
				"botKind": b.GetString("botKind"),
			})
		}
		return e.JSON(http.StatusOK, map[string]any{"bots": out})
	})

	// POST /api/leagues/{id}/bots/add  { "userId": "..." } — add a bot account
	// to the league. Owner-only, and the target must actually be a bot user so
	// owners can't conscript arbitrary people into their league.
	g.POST("/{id}/bots/add", func(e *core.RequestEvent) error {
		lg, err := ownedOpenLeague(app, e, e.Request.PathValue("id"))
		if err != nil {
			return err
		}
		var body struct {
			UserID string `json:"userId"`
		}
		if err := e.BindBody(&body); err != nil {
			return bad(e, http.StatusBadRequest, err.Error())
		}
		if body.UserID == "" {
			return bad(e, http.StatusBadRequest, "userId required")
		}
		u, err := app.FindRecordById("users", body.UserID)
		if err != nil {
			return bad(e, http.StatusNotFound, "user not found")
		}
		if u.GetString("role") != "bot" {
			return bad(e, http.StatusBadRequest, "only bot accounts can be added this way")
		}
		if existing, _ := app.FindFirstRecordByFilter("pool_members",
			"pool = {:l} && user = {:u}",
			map[string]any{"l": lg.Id, "u": body.UserID}); existing != nil {
			return e.JSON(http.StatusOK, map[string]any{"ok": true, "already": true})
		}
		if err := addMember(app, lg.Id, body.UserID, "member"); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})
}

func addMember(app core.App, leagueID, userID, role string) error {
	col, err := app.FindCollectionByNameOrId("pool_members")
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("pool", leagueID)
	rec.Set("user", userID)
	rec.Set("role", role)
	return app.Save(rec)
}

// ensureGlobal idempotently creates the "Global" league (owner left empty so
// no one can update/delete it via REST). Returns the league id.
func ensureGlobal(app core.App) (string, error) {
	if rec, err := app.FindFirstRecordByFilter("pools",
		"inviteCode = {:c}", map[string]any{"c": GlobalInviteCode}); err == nil {
		return rec.Id, nil
	}
	col, err := app.FindCollectionByNameOrId("pools")
	if err != nil {
		return "", err
	}
	def, _ := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
	rec := core.NewRecord(col)
	rec.Set("name", "Global")
	rec.Set("inviteCode", GlobalInviteCode)
	if def != nil {
		rec.Set("scoringConfig", def.Id)
	}
	if err := app.Save(rec); err != nil {
		return "", err
	}
	return rec.Id, nil
}

// ensureGlobalMember adds the user to the Global league if not already a member.
func ensureGlobalMember(app core.App, userID string) error {
	leagueID, err := ensureGlobal(app)
	if err != nil {
		return err
	}
	if existing, _ := app.FindFirstRecordByFilter("pool_members",
		"pool = {:l} && user = {:u}",
		map[string]any{"l": leagueID, "u": userID}); existing != nil {
		return nil
	}
	return addMember(app, leagueID, userID, "member")
}

// backfillGlobal ensures every existing user is a member of the Global league.
// Cheap on subsequent boots: the per-user membership check short-circuits.
func backfillGlobal(app core.App) error {
	if _, err := ensureGlobal(app); err != nil {
		return err
	}
	users, err := app.FindRecordsByFilter("users", "id != ''", "", 0, 0)
	if err != nil {
		return err
	}
	for _, u := range users {
		if err := ensureGlobalMember(app, u.Id); err != nil {
			return err
		}
	}
	return nil
}
