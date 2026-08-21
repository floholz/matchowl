package tournaments

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/floholz/matchowl/internal/users"
)

const compCollection = "competitions"

// CompetitionView is the public shape of a competition, embedded in
// tournament payloads and the catalog.
func CompetitionView(r *core.Record) map[string]any {
	if r == nil {
		return nil
	}
	return map[string]any{
		"id":        r.Id,
		"key":       r.GetString("key"),
		"name":      r.GetString("name"),
		"shortName": r.GetString("shortName"),
		"country":   r.GetString("country"),
		"teamKind":  r.GetString("teamKind"),
		"logo":      r.GetString("logo"), // filename; client builds /api/files/... URL
		// Admin blurb; empty → the client derives one from the structure.
		"description":       r.GetString("description"),
		"apiFootballLeague": r.GetInt("apiFootballLeague"),
	}
}

// ByLeagueID finds the competition mapped to an API-Football league id.
func ByLeagueID(app core.App, leagueID int) (*core.Record, error) {
	return app.FindFirstRecordByFilter(compCollection,
		"apiFootballLeague = {:l}", map[string]any{"l": leagueID})
}

// LeagueLogoURL is the provider's stable (public, key-less) league badge.
func LeagueLogoURL(leagueID int) string {
	return fmt.Sprintf("https://media.api-sports.io/football/leagues/%d.png", leagueID)
}

var compKeyJunk = regexp.MustCompile(`[^a-z0-9]+`)

// KeyFromName makes a competition key from a display name ("UEFA Champions
// League" → "uefa-champions-league"), de-duplicated against existing keys.
func KeyFromName(app core.App, name string) string {
	base := strings.Trim(compKeyJunk.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(base) < 2 {
		base = "competition"
	}
	if len(base) > 32 {
		base = strings.Trim(base[:32], "-")
	}
	key := base
	for i := 2; ; i++ {
		if _, err := app.FindFirstRecordByFilter(compCollection, "key = {:k}", map[string]any{"k": key}); err != nil {
			return key
		}
		suffix := fmt.Sprintf("-%d", i)
		key = base
		if len(key)+len(suffix) > 32 {
			key = key[:32-len(suffix)]
		}
		key += suffix
	}
}

// LeagueInfo is what the importer knows about a provider league.
type LeagueInfo struct {
	ID       int
	Name     string
	Country  string
	TeamKind string // national | club
	LogoURL  string
}

// EnsureForLeague returns the competition mapped to the league, creating
// one from the league's catalog entry when none exists yet (the admin can
// rename it afterwards). The logo is fetched best-effort, outside any
// transaction the caller may hold.
func EnsureForLeague(app core.App, l LeagueInfo) (*core.Record, error) {
	if c, err := ByLeagueID(app, l.ID); err == nil {
		return c, nil
	}
	col, err := app.FindCollectionByNameOrId(compCollection)
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(col)
	rec.Set("key", KeyFromName(app, l.Name))
	rec.Set("name", strings.TrimSpace(l.Name))
	rec.Set("country", strings.TrimSpace(l.Country))
	if l.TeamKind != "club" {
		l.TeamKind = "national"
	}
	rec.Set("teamKind", l.TeamKind)
	rec.Set("apiFootballLeague", l.ID)
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// AttachLogo downloads the league badge into competitions.logo when the
// record has none yet. Best-effort; returns whether a logo was stored.
func AttachLogo(ctx context.Context, app core.App, rec *core.Record, url string) bool {
	if rec.GetString("logo") != "" || url == "" {
		return false
	}
	fctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	f, err := filesystem.NewFileFromURL(fctx, url)
	if err != nil {
		log.Printf("[competitions] logo %s: %v", rec.GetString("key"), err)
		return false
	}
	rec.Set("logo", f)
	if err := app.Save(rec); err != nil {
		log.Printf("[competitions] save logo %s: %v", rec.GetString("key"), err)
		return false
	}
	return true
}

// competitionOf loads a tournament's competition record ("" relation → nil).
func competitionOf(app core.App, t *core.Record) *core.Record {
	cid := t.GetString("competition")
	if cid == "" {
		return nil
	}
	c, err := app.FindRecordById(compCollection, cid)
	if err != nil {
		return nil
	}
	return c
}

type compPayload struct {
	Key               *string `json:"key"`
	Name              *string `json:"name"`
	ShortName         *string `json:"shortName"`
	Country           *string `json:"country"`
	TeamKind          *string `json:"teamKind"`
	Description       *string `json:"description"`
	APIFootballLeague *int    `json:"apiFootballLeague"`
}

func (p *compPayload) applyTo(rec *core.Record) error {
	if p.Key != nil {
		k := strings.TrimSpace(*p.Key)
		if !slugRe.MatchString(k) {
			return apis.NewBadRequestError("key must be 2-32 chars of a-z 0-9 -", nil)
		}
		rec.Set("key", k)
	}
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n == "" || len(n) > 100 {
			return apis.NewBadRequestError("name required (max 100 chars)", nil)
		}
		rec.Set("name", n)
	}
	if p.ShortName != nil {
		rec.Set("shortName", strings.TrimSpace(*p.ShortName))
	}
	if p.Country != nil {
		rec.Set("country", strings.TrimSpace(*p.Country))
	}
	if p.TeamKind != nil {
		if *p.TeamKind != "national" && *p.TeamKind != "club" {
			return apis.NewBadRequestError("teamKind must be national or club", nil)
		}
		rec.Set("teamKind", *p.TeamKind)
	}
	if p.Description != nil {
		d := strings.TrimSpace(*p.Description)
		if len(d) > 500 {
			return apis.NewBadRequestError("description max 500 chars", nil)
		}
		rec.Set("description", d)
	}
	if p.APIFootballLeague != nil {
		rec.Set("apiFootballLeague", *p.APIFootballLeague)
	}
	return nil
}

// registerCompetitions wires the admin CRUD for competitions and the
// clone-season action. Logo uploads go through the PB record API by a
// superuser (file fields need multipart handling the dashboard already has).
func registerCompetitions(app core.App, se *core.ServeEvent) {
	g := se.Router.Group("/api/admin/competitions")
	g.Bind(apis.RequireAuth())
	g.BindFunc(func(e *core.RequestEvent) error {
		if e.Auth == nil || !users.IsAdmin(e.Auth) {
			return apis.NewForbiddenError("admin only", nil)
		}
		return e.Next()
	})

	g.GET("", func(e *core.RequestEvent) error {
		recs, err := app.FindRecordsByFilter(compCollection, "id != ''", "name", 0, 0)
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, CompetitionView(r))
		}
		return e.JSON(http.StatusOK, map[string]any{"competitions": out})
	})

	g.POST("", func(e *core.RequestEvent) error {
		var body compPayload
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if body.Key == nil || body.Name == nil || body.TeamKind == nil {
			return apis.NewBadRequestError("key, name and teamKind are required", nil)
		}
		col, err := app.FindCollectionByNameOrId(compCollection)
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		if err := body.applyTo(rec); err != nil {
			return err
		}
		if err := app.Save(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, CompetitionView(rec))
	})

	g.POST("/{id}", func(e *core.RequestEvent) error {
		rec, err := app.FindRecordById(compCollection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such competition", nil)
		}
		var body compPayload
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if err := body.applyTo(rec); err != nil {
			return err
		}
		if err := app.Save(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, CompetitionView(rec))
	})

	// POST /{id}/logo — (re)fetch the league badge from API-Football by the
	// competition's league id. Replaces an existing logo.
	g.POST("/{id}/logo", func(e *core.RequestEvent) error {
		rec, err := app.FindRecordById(compCollection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such competition", nil)
		}
		lid := rec.GetInt("apiFootballLeague")
		if lid == 0 {
			return apis.NewBadRequestError("competition has no apiFootballLeague id", nil)
		}
		rec.Set("logo", nil)
		if !AttachLogo(e.Request.Context(), app, rec, LeagueLogoURL(lid)) {
			return e.JSON(http.StatusBadGateway, map[string]string{"error": "could not fetch the logo"})
		}
		return e.JSON(http.StatusOK, CompetitionView(rec))
	})

	// DELETE only when no tournament references it.
	g.DELETE("/{id}", func(e *core.RequestEvent) error {
		rec, err := app.FindRecordById(compCollection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such competition", nil)
		}
		if _, err := app.FindFirstRecordByFilter(collection,
			"competition = {:c}", map[string]any{"c": rec.Id}); err == nil {
			return apis.NewBadRequestError("competition still has tournaments", nil)
		}
		if err := app.Delete(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})
}

// registerClone wires the clone-season action on the admin tournaments
// group: copies structure, sync, forecast spec and competition from an
// existing tournament into a new draft season.
func registerClone(app core.App, g *router.RouterGroup[*core.RequestEvent]) {
	g.POST("/{id}/clone", func(e *core.RequestEvent) error {
		src, err := app.FindRecordById(collection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		var body struct {
			Slug        string `json:"slug"`
			Name        string `json:"name"`
			ShortName   string `json:"shortName"`
			ExtIDPrefix string `json:"extIdPrefix"`
			Season      int    `json:"season"`
			StartsAt    string `json:"startsAt"`
			EndsAt      string `json:"endsAt"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if !slugRe.MatchString(body.Slug) {
			return apis.NewBadRequestError("slug must be 2-32 chars of a-z 0-9 -", nil)
		}
		if body.ExtIDPrefix == "" || len(body.ExtIDPrefix) > 16 {
			return apis.NewBadRequestError("extIdPrefix required (max 16 chars)", nil)
		}
		col, err := app.FindCollectionByNameOrId(collection)
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		rec.Set("slug", body.Slug)
		rec.Set("name", strings.TrimSpace(body.Name))
		if rec.GetString("name") == "" {
			rec.Set("name", src.GetString("name"))
		}
		rec.Set("shortName", strings.TrimSpace(body.ShortName))
		rec.Set("status", StatusDraft)
		rec.Set("extIdPrefix", body.ExtIDPrefix)
		rec.Set("structure", src.Get("structure"))
		rec.Set("forecastSpec", src.Get("forecastSpec"))
		rec.Set("competition", src.GetString("competition"))
		rec.Set("scoringConfig", src.GetString("scoringConfig"))
		rec.Set("startsAt", strings.TrimSpace(body.StartsAt))
		rec.Set("endsAt", strings.TrimSpace(body.EndsAt))
		if sync, err := SyncOf(src); err == nil {
			if body.Season != 0 {
				sync.Season = body.Season
			}
			if raw, err := json.Marshal(sync); err == nil {
				rec.Set("sync", string(raw))
			}
		}
		if err := app.Save(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, view(rec))
	})
}
