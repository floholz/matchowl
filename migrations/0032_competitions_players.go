package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Feed rework, step 1 (PLAN-feed.md): competitions as a first-class entity
// linking seasons, and explicit per-user tournament participation.
//
//   - `competitions`: key/name/country/teamKind/logo/apiFootballLeague; the
//     provider's stable league id lives here, the season on the tournament.
//     Seeds the "world-cup" competition and links wc2026 to it.
//   - `tournaments.competition` relation (optional).
//   - `tournament_players`: who plays which tournament (source manual|auto).
//     Every existing user is backfilled as a wc2026 player — they all played
//     it under the implicit single-tournament model.
//   - `teams.clubKey`: optional stable club identity across season rows.
const (
	nCompetitions = "competitions"
	nPlayers      = "tournament_players"
)

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId(nCompetitions); err == nil {
			return nil // already migrated
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		tcol, err := app.FindCollectionByNameOrId(nTournaments)
		if err != nil {
			return err
		}

		// ---- competitions ----
		comps := core.NewBaseCollection(nCompetitions)
		comps.ListRule = ptr("")
		comps.ViewRule = ptr("")
		comps.Fields.Add(&core.TextField{Name: "key", Required: true, Max: 32})
		comps.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 100})
		comps.Fields.Add(&core.TextField{Name: "shortName", Max: 32})
		comps.Fields.Add(&core.TextField{Name: "country", Max: 32})
		comps.Fields.Add(&core.SelectField{
			Name: "teamKind", Required: true, MaxSelect: 1,
			Values: []string{"national", "club"},
		})
		comps.Fields.Add(&core.FileField{
			Name: "logo", MaxSelect: 1, MaxSize: 1 << 20,
			MimeTypes: []string{"image/png", "image/svg+xml", "image/webp", "image/jpeg"},
		})
		comps.Fields.Add(&core.NumberField{Name: "apiFootballLeague", OnlyInt: true})
		comps.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		comps.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		comps.AddIndex("idx_competitions_key", true, "key", "")
		if err := app.Save(comps); err != nil {
			return err
		}

		wc := core.NewRecord(comps)
		wc.Set("key", "world-cup")
		wc.Set("name", "FIFA World Cup")
		wc.Set("shortName", "World Cup")
		wc.Set("teamKind", "national")
		wc.Set("apiFootballLeague", 1)
		if err := app.Save(wc); err != nil {
			return err
		}

		// ---- tournaments.competition ----
		tcol.Fields.Add(&core.RelationField{Name: "competition", CollectionId: comps.Id, MaxSelect: 1})
		if err := app.Save(tcol); err != nil {
			return err
		}
		if t, err := app.FindFirstRecordByFilter(nTournaments, "slug = 'wc2026'"); err == nil {
			t.Set("competition", wc.Id)
			if err := app.Save(t); err != nil {
				return err
			}
		}

		// ---- tournament_players ----
		tp := core.NewBaseCollection(nPlayers)
		own := "@request.auth.id = user"
		tp.ListRule = ptr(own)
		tp.ViewRule = ptr(own) // writes go through the Play/Leave endpoints
		tp.Fields.Add(&core.RelationField{Name: "tournament", CollectionId: tcol.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		tp.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		tp.Fields.Add(&core.SelectField{
			Name: "source", Required: true, MaxSelect: 1,
			Values: []string{"manual", "auto"},
		})
		tp.Fields.Add(&core.AutodateField{Name: "joinedAt", OnCreate: true})
		tp.AddIndex("idx_tp_tournament_user", true, "tournament, user", "")
		if err := app.Save(tp); err != nil {
			return err
		}

		// Backfill: everyone played wc2026 under the single-tournament model.
		if t, err := app.FindFirstRecordByFilter(nTournaments, "slug = 'wc2026'"); err == nil {
			all, err := app.FindRecordsByFilter("users", "id != ''", "", 0, 0)
			if err != nil {
				return err
			}
			for _, u := range all {
				rec := core.NewRecord(tp)
				rec.Set("tournament", t.Id)
				rec.Set("user", u.Id)
				rec.Set("source", "auto")
				if err := app.Save(rec); err != nil {
					return err
				}
			}
		}

		// ---- teams.clubKey ----
		teams, err := app.FindCollectionByNameOrId("teams")
		if err != nil {
			return err
		}
		teams.Fields.Add(&core.TextField{Name: "clubKey", Max: 32})
		return app.Save(teams)
	}, func(app core.App) error {
		for _, name := range []string{nPlayers, nCompetitions} {
			if c, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(c); err != nil {
					return err
				}
			}
		}
		if tcol, err := app.FindCollectionByNameOrId(nTournaments); err == nil {
			tcol.Fields.RemoveByName("competition")
			if err := app.Save(tcol); err != nil {
				return err
			}
		}
		if teams, err := app.FindCollectionByNameOrId("teams"); err == nil {
			teams.Fields.RemoveByName("clubKey")
			return app.Save(teams)
		}
		return nil
	})
}
