package tournaments

import (
	"net/http"
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/football"
	"github.com/floholz/matchowl/internal/users"
)

// The teams registry (2026-09-28): teams are one row per tournament, so a
// club or nation shows up once per season it plays in. The registry
// groups those rows into one entry per team — by provider id when the
// rows carry one, else by the club key or the normalized name — with the
// seasons it plays in, and lets an admin fix code, flag and name across
// all of them at once.

type registryEntry struct {
	TeamID     string `json:"teamId"`
	Tournament string `json:"tournament"` // id
	Slug       string `json:"slug"`
	Season     string `json:"season"` // tournament name
	Status     string `json:"status"`
	Group      string `json:"group"`
	FifaCode   string `json:"fifaCode"`
	ISO2       string `json:"iso2"`
	Logo       string `json:"logo"`
	Name       string `json:"name"`
}

type registryTeam struct {
	Key        string          `json:"key"`
	Name       string          `json:"name"`
	FifaCode   string          `json:"fifaCode"`
	ISO2       string          `json:"iso2"`
	ClubKey    string          `json:"clubKey"`
	Provider   string          `json:"provider"`
	ProviderID int             `json:"providerId"`
	Logo       string          `json:"logo"`  // "<teamId>/<file>" of the first row with a crest
	Mixed      []string        `json:"mixed"` // fields that differ across the rows
	Entries    []registryEntry `json:"entries"`
}

// registryKey is what makes two team rows the same team.
func registryKey(tm *core.Record) string {
	if p, id := tm.GetString("provider"), tm.GetInt("providerId"); p != "" && id != 0 {
		return p + ":" + tm.GetString("providerId")
	}
	if ck := tm.GetString("clubKey"); ck != "" {
		return "club:" + ck
	}
	if iso := tm.GetString("iso2"); iso != "" {
		return "nation:" + iso
	}
	return "name:" + football.NormalizeName(tm.GetString("name"))
}

func registry(app core.App) ([]registryTeam, error) {
	teams, err := app.FindRecordsByFilter("teams", "", "name", 0, 0)
	if err != nil {
		return nil, err
	}
	tours := map[string]*core.Record{}
	groupOf := map[string]string{} // team id → letter
	byKey := map[string]*registryTeam{}
	for _, tm := range teams {
		tid := tm.GetString("tournament")
		t := tours[tid]
		if t == nil {
			if t, err = app.FindRecordById(collection, tid); err != nil {
				continue
			}
			tours[tid] = t
			groups, _ := app.FindRecordsByFilter("tournament_groups", "tournament = {:t}", "", 0, 0, map[string]any{"t": tid})
			for _, g := range groups {
				for _, id := range g.GetStringSlice("teams") {
					groupOf[id] = g.GetString("letter")
				}
			}
		}
		key := registryKey(tm)
		rt := byKey[key]
		if rt == nil {
			rt = &registryTeam{
				Key: key, Name: tm.GetString("name"), FifaCode: tm.GetString("fifaCode"), ISO2: tm.GetString("iso2"),
				ClubKey: tm.GetString("clubKey"), Provider: tm.GetString("provider"), ProviderID: tm.GetInt("providerId"),
				Mixed: []string{}, Entries: []registryEntry{},
			}
			byKey[key] = rt
		}
		if rt.Logo == "" && tm.GetString("logo") != "" {
			rt.Logo = tm.Id + "/" + tm.GetString("logo")
		}
		mixed := func(field string) {
			for _, m := range rt.Mixed {
				if m == field {
					return
				}
			}
			rt.Mixed = append(rt.Mixed, field)
		}
		if tm.GetString("name") != rt.Name {
			mixed("name")
		}
		if tm.GetString("fifaCode") != rt.FifaCode {
			mixed("fifaCode")
		}
		if tm.GetString("iso2") != rt.ISO2 {
			mixed("iso2")
		}
		rt.Entries = append(rt.Entries, registryEntry{
			TeamID: tm.Id, Tournament: tid, Slug: t.GetString("slug"), Season: t.GetString("name"),
			Status: t.GetString("status"), Group: groupOf[tm.Id], FifaCode: tm.GetString("fifaCode"),
			ISO2: tm.GetString("iso2"), Logo: tm.GetString("logo"), Name: tm.GetString("name"),
		})
	}
	out := make([]registryTeam, 0, len(byKey))
	for _, rt := range byKey {
		sort.SliceStable(rt.Entries, func(i, j int) bool { return rt.Entries[i].Season > rt.Entries[j].Season })
		out = append(out, *rt)
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// registerTeams wires GET /api/admin/teams and POST /api/admin/teams/{id}.
func registerTeams(app core.App, se *core.ServeEvent) {
	g := se.Router.Group("/api/admin/teams")
	g.Bind(apis.RequireAuth())
	g.BindFunc(func(e *core.RequestEvent) error {
		if e.Auth == nil || !users.IsAdmin(e.Auth) {
			return apis.NewForbiddenError("admin only", nil)
		}
		return e.Next()
	})
	g.GET("", func(e *core.RequestEvent) error {
		teams, err := registry(app)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"teams": teams})
	})
	// POST /api/admin/teams/{id} { name?, fifaCode?, iso2?, applyToAll? }
	// — edits one row; applyToAll carries the change to every row of the
	// same team (registry key).
	g.POST("/{id}", func(e *core.RequestEvent) error {
		tm, err := app.FindRecordById("teams", e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such team", nil)
		}
		var body struct {
			Name       *string `json:"name"`
			FifaCode   *string `json:"fifaCode"`
			ISO2       *string `json:"iso2"`
			ApplyToAll bool    `json:"applyToAll"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if body.FifaCode != nil {
			c := strings.ToUpper(strings.TrimSpace(*body.FifaCode))
			if len(c) < 2 || len(c) > 3 {
				return apis.NewBadRequestError("code must be 2–3 characters", nil)
			}
			*body.FifaCode = c
		}
		if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
			return apis.NewBadRequestError("name must not be empty", nil)
		}
		rows := []*core.Record{tm}
		if body.ApplyToAll {
			key := registryKey(tm)
			all, _ := app.FindRecordsByFilter("teams", "", "", 0, 0)
			for _, o := range all {
				if o.Id != tm.Id && registryKey(o) == key {
					rows = append(rows, o)
				}
			}
		}
		n := 0
		err = app.RunInTransaction(func(tx core.App) error {
			for _, r := range rows {
				if body.Name != nil {
					r.Set("name", strings.TrimSpace(*body.Name))
				}
				if body.FifaCode != nil {
					r.Set("fifaCode", *body.FifaCode)
				}
				if body.ISO2 != nil {
					r.Set("iso2", strings.ToLower(strings.TrimSpace(*body.ISO2)))
				}
				if err := tx.Save(r); err != nil {
					return apis.NewBadRequestError("could not save "+r.GetString("name")+": "+err.Error(), nil)
				}
				n++
			}
			return nil
		})
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"updated": n})
	})
}
