package tournaments

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
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
	}
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
			v := CompetitionView(r)
			v["apiFootballLeague"] = r.GetInt("apiFootballLeague")
			out = append(out, v)
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
		// forecastSpec is copied here too once phase 4 adds the field.
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
