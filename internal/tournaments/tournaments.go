package tournaments

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/users"
)

const collection = "tournaments"

// BySlug returns the tournament record with the given slug.
func BySlug(app core.App, slug string) (*core.Record, error) {
	return app.FindFirstRecordByFilter(collection, "slug = {:s}", map[string]any{"s": slug})
}

// StructureOf parses, normalizes and returns a record's structure JSON.
func StructureOf(rec *core.Record) (*Structure, error) {
	s := &Structure{}
	if err := rec.UnmarshalJSONField("structure", s); err != nil {
		return nil, err
	}
	s.Normalize()
	return s, nil
}

// structureView is the structure as the app should see it: parsed and
// normalized (default points, default zones), falling back to the raw JSON
// when it doesn't parse.
func structureView(r *core.Record) any {
	if st, err := StructureOf(r); err == nil {
		return st
	}
	return r.Get("structure")
}

// SyncOf parses and returns a record's sync JSON.
func SyncOf(rec *core.Record) (*Sync, error) {
	s := &Sync{}
	if err := rec.UnmarshalJSONField("sync", s); err != nil {
		return nil, err
	}
	return s, nil
}

// All returns every tournament, newest startsAt first (draft included —
// callers filter).
func All(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter(collection, "id != ''", "-startsAt,-created", 0, 0)
}

// Active returns tournaments whose results should be syncing.
func Active(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter(collection, "status = {:s}", "-startsAt", 0, 0,
		map[string]any{"s": StatusActive})
}

var statusRank = map[string]int{
	StatusActive:   0,
	StatusUpcoming: 1,
	StatusFinished: 2,
	StatusArchived: 3,
}

// Current returns the tournament users interact with by default: an active
// one first, else the next upcoming, else the most recently finished/archived.
func Current(app core.App) (*core.Record, error) {
	recs, err := All(app)
	if err != nil {
		return nil, err
	}
	var best *core.Record
	for _, r := range recs {
		if r.GetString("status") == StatusDraft {
			continue
		}
		if best == nil || rankLess(r, best) {
			best = r
		}
	}
	if best == nil {
		return nil, sql.ErrNoRows
	}
	return best, nil
}

// rankLess reports whether a should be preferred over b as "current".
func rankLess(a, b *core.Record) bool {
	ra, rb := statusRank[a.GetString("status")], statusRank[b.GetString("status")]
	if ra != rb {
		return ra < rb
	}
	at, bt := a.GetDateTime("startsAt").Time(), b.GetDateTime("startsAt").Time()
	if a.GetString("status") == StatusUpcoming {
		return at.Before(bt) // soonest upcoming
	}
	return at.After(bt) // latest otherwise
}

func view(r *core.Record) map[string]any {
	spec, _ := ForecastSpecOf(r)
	return map[string]any{
		"id":           r.Id,
		"slug":         r.GetString("slug"),
		"name":         r.GetString("name"),
		"shortName":    r.GetString("shortName"),
		"status":       r.GetString("status"),
		"startsAt":     r.GetString("startsAt"),
		"endsAt":       r.GetString("endsAt"),
		"structure":    structureView(r),
		"forecastSpec": spec,
		"extIdPrefix":  r.GetString("extIdPrefix"),
	}
}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,31}$`)

// Payload is the admin create/update body. Nil pointers / empty raw JSON
// mean "leave unchanged" so the same payload serves create and update.
type Payload struct {
	Slug          *string         `json:"slug"`
	Name          *string         `json:"name"`
	ShortName     *string         `json:"shortName"`
	Status        *string         `json:"status"`
	StartsAt      *string         `json:"startsAt"`
	EndsAt        *string         `json:"endsAt"`
	Structure     json.RawMessage `json:"structure"`
	Sync          json.RawMessage `json:"sync"`
	ForecastSpec  json.RawMessage `json:"forecastSpec"`
	ExtIDPrefix   *string         `json:"extIdPrefix"`
	ScoringConfig *string         `json:"scoringConfig"`
	Competition   *string         `json:"competition"` // competitions record id ("" clears)
}

// applyTo validates the provided fields and writes them onto the record.
func (p *Payload) applyTo(app core.App, rec *core.Record) error {
	if p.Slug != nil {
		s := strings.TrimSpace(*p.Slug)
		if !slugRe.MatchString(s) {
			return apis.NewBadRequestError("slug must be 2-32 chars of a-z 0-9 -", nil)
		}
		rec.Set("slug", s)
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
	if p.Status != nil {
		ok := false
		for _, s := range Statuses {
			if s == *p.Status {
				ok = true
			}
		}
		if !ok {
			return apis.NewBadRequestError("unknown status", nil)
		}
		rec.Set("status", *p.Status)
	}
	if p.StartsAt != nil {
		rec.Set("startsAt", strings.TrimSpace(*p.StartsAt))
	}
	if p.EndsAt != nil {
		rec.Set("endsAt", strings.TrimSpace(*p.EndsAt))
	}
	if len(p.Structure) > 0 {
		s := &Structure{}
		if err := json.Unmarshal(p.Structure, s); err != nil {
			return apis.NewBadRequestError("structure: "+err.Error(), nil)
		}
		s.Normalize()
		if err := s.Validate(); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		norm, err := json.Marshal(s)
		if err != nil {
			return err
		}
		rec.Set("structure", string(norm))
	}
	if len(p.Sync) > 0 {
		s := &Sync{}
		if err := json.Unmarshal(p.Sync, s); err != nil {
			return apis.NewBadRequestError("sync: "+err.Error(), nil)
		}
		if err := s.Validate(); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		rec.Set("sync", string(p.Sync))
	}
	if len(p.ForecastSpec) > 0 {
		f := &ForecastSpec{}
		if err := json.Unmarshal(p.ForecastSpec, f); err != nil {
			return apis.NewBadRequestError("forecastSpec: "+err.Error(), nil)
		}
		rec.Set("forecastSpec", string(p.ForecastSpec))
	}
	if p.ExtIDPrefix != nil {
		pre := strings.TrimSpace(*p.ExtIDPrefix)
		if pre == "" || len(pre) > 16 {
			return apis.NewBadRequestError("extIdPrefix required (max 16 chars)", nil)
		}
		rec.Set("extIdPrefix", pre)
	}
	if p.ScoringConfig != nil {
		if *p.ScoringConfig != "" {
			if _, err := app.FindRecordById("scoring_configs", *p.ScoringConfig); err != nil {
				return apis.NewBadRequestError("unknown scoringConfig", nil)
			}
		}
		rec.Set("scoringConfig", *p.ScoringConfig)
	}
	if p.Competition != nil {
		if *p.Competition != "" {
			if _, err := app.FindRecordById("competitions", *p.Competition); err != nil {
				return apis.NewBadRequestError("unknown competition", nil)
			}
		}
		rec.Set("competition", *p.Competition)
	}
	return nil
}

// Create validates the payload and inserts a new tournament. It lands as
// draft unless the payload names a status, and falls back to the default
// scoring config when none is given (a tournament without one can't score).
func Create(app core.App, body *Payload) (*core.Record, error) {
	if body.Slug == nil || body.Name == nil || len(body.Structure) == 0 || body.ExtIDPrefix == nil {
		return nil, apis.NewBadRequestError("slug, name, structure and extIdPrefix are required", nil)
	}
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(col)
	rec.Set("status", StatusDraft)
	if err := body.applyTo(app, rec); err != nil {
		return nil, err
	}
	if rec.GetString("scoringConfig") == "" {
		if def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true"); err == nil {
			rec.Set("scoringConfig", def.Id)
		}
	}
	if err := validateSpec(rec); err != nil {
		return nil, err
	}
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// validateSpec checks the record's forecastSpec against its structure and
// persists the normalized form (teamset counts filled from zones).
func validateSpec(rec *core.Record) error {
	st, err := StructureOf(rec)
	if err != nil {
		return nil // structure invalid/absent is caught elsewhere
	}
	spec, _ := ForecastSpecOf(rec)
	if err := spec.Validate(st); err != nil {
		return apis.NewBadRequestError(err.Error(), nil)
	}
	norm, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	rec.Set("forecastSpec", string(norm))
	return nil
}

// Register wires the tournament endpoints: public list/detail for the SPA
// and an admin CRUD group. Fixture seeding for new tournaments is a separate
// concern (internal/seed).
func Register(app core.App, se *core.ServeEvent) {
	BackfillLogos(app)

	// GET /api/tournaments — non-draft tournaments plus the current pick,
	// each with its competition embedded (the catalog groups by it).
	se.Router.GET("/api/tournaments", func(e *core.RequestEvent) error {
		recs, err := All(app)
		if err != nil {
			return err
		}
		comps := map[string]map[string]any{}
		out := make([]map[string]any, 0, len(recs))
		var current *core.Record
		// Drafts are hidden from players; admins see them (with a draft
		// pill) so they can preview a tournament before publishing it. A
		// draft never becomes "current".
		isAdmin := e.Auth != nil && users.IsAdmin(e.Auth)
		for _, r := range recs {
			draft := r.GetString("status") == StatusDraft
			if draft && !isAdmin {
				continue
			}
			v := view(r)
			if cid := r.GetString("competition"); cid != "" {
				if _, ok := comps[cid]; !ok {
					comps[cid] = CompetitionView(competitionOf(app, r))
				}
				v["competition"] = comps[cid]
			}
			out = append(out, v)
			if !draft && (current == nil || rankLess(r, current)) {
				current = r
			}
		}
		sort.SliceStable(out, func(i, j int) bool {
			return out[i]["startsAt"].(string) > out[j]["startsAt"].(string)
		})
		res := map[string]any{"tournaments": out}
		if current != nil {
			res["current"] = current.Id
		}
		return e.JSON(http.StatusOK, res)
	})

	// GET /api/tournaments/{slug} — one tournament (draft: admin only).
	se.Router.GET("/api/tournaments/{slug}", func(e *core.RequestEvent) error {
		rec, err := BySlug(app, e.Request.PathValue("slug"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		if rec.GetString("status") == StatusDraft &&
			(e.Auth == nil || !users.IsAdmin(e.Auth)) {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		v := view(rec)
		if c := competitionOf(app, rec); c != nil {
			v["competition"] = CompetitionView(c)
		}
		return e.JSON(http.StatusOK, v)
	})

	g := se.Router.Group("/api/admin/tournaments")
	g.Bind(apis.RequireAuth())
	g.BindFunc(func(e *core.RequestEvent) error {
		if e.Auth == nil || !users.IsAdmin(e.Auth) {
			return apis.NewForbiddenError("admin only", nil)
		}
		return e.Next()
	})

	// GET /api/admin/tournaments — everything, drafts included.
	g.GET("", func(e *core.RequestEvent) error {
		recs, err := All(app)
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			v := view(r)
			v["sync"] = r.Get("sync")
			v["scoringConfig"] = r.GetString("scoringConfig")
			v["competition"] = r.GetString("competition")
			nTeams, _ := app.CountRecords("teams", dbx.HashExp{"tournament": r.Id})
			nMatches, _ := app.CountRecords("matches", dbx.HashExp{"tournament": r.Id})
			nPlayers, _ := app.CountRecords("tournament_players", dbx.HashExp{"tournament": r.Id})
			v["teams"] = nTeams
			v["matches"] = nMatches
			v["players"] = nPlayers
			out = append(out, v)
		}
		return e.JSON(http.StatusOK, map[string]any{"tournaments": out})
	})

	// POST /api/admin/tournaments — create (as draft unless status given).
	g.POST("", func(e *core.RequestEvent) error {
		var body Payload
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		rec, err := Create(app, &body)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, view(rec))
	})

	// POST /api/admin/tournaments/{id} — update.
	g.POST("/{id}", func(e *core.RequestEvent) error {
		rec, err := app.FindRecordById(collection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		var body Payload
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if err := body.applyTo(app, rec); err != nil {
			return err
		}
		if err := validateSpec(rec); err != nil {
			return err
		}
		if err := app.Save(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, view(rec))
	})

	registerClone(app, g)

	// DELETE /api/admin/tournaments/{id} — draft tournaments only. Deleting
	// a played tournament would cascade through teams/matches into user data.
	g.DELETE("/{id}", func(e *core.RequestEvent) error {
		rec, err := app.FindRecordById(collection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		if rec.GetString("status") != StatusDraft {
			return apis.NewBadRequestError("only draft tournaments can be deleted", nil)
		}
		if err := app.Delete(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})

	registerCompetitions(app, se)
}
