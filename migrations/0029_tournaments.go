package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/floholz/matchowl/internal/tournaments"
)

const nTournaments = "tournaments"

// Multi-tournament rework, step 1: the tournaments root collection, plus the
// `wc2026` record every existing row is scoped to in 0030. Its structure/sync
// JSON reproduce exactly what the code hardcoded before the rework, and the
// record is created archived — WC 2026 is over.
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId(nTournaments); err == nil {
			return nil // already migrated
		}
		sc, err := app.FindCollectionByNameOrId("scoring_configs")
		if err != nil {
			return err
		}

		col := core.NewBaseCollection(nTournaments)
		col.ListRule = ptr("")
		col.ViewRule = ptr("")
		col.Fields.Add(&core.TextField{Name: "slug", Required: true, Max: 32})
		col.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 100})
		col.Fields.Add(&core.TextField{Name: "shortName", Max: 32})
		col.Fields.Add(&core.SelectField{
			Name: "status", Required: true, MaxSelect: 1,
			Values: tournaments.Statuses,
		})
		col.Fields.Add(&core.DateField{Name: "startsAt"})
		col.Fields.Add(&core.DateField{Name: "endsAt"})
		col.Fields.Add(&core.JSONField{Name: "structure", MaxSize: 20000})
		col.Fields.Add(&core.JSONField{Name: "sync", MaxSize: 4000})
		col.Fields.Add(&core.TextField{Name: "extIdPrefix", Required: true, Max: 16})
		col.Fields.Add(&core.RelationField{Name: "scoringConfig", CollectionId: sc.Id, MaxSelect: 1})
		col.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		col.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		col.AddIndex("idx_tournaments_slug", true, "slug", "")
		if err := app.Save(col); err != nil {
			return err
		}

		rec := core.NewRecord(col)
		rec.Set("slug", tournaments.WC2026Slug)
		rec.Set("name", "FIFA World Cup 2026")
		rec.Set("shortName", "WC 2026")
		rec.Set("status", tournaments.StatusArchived)
		rec.Set("structure", tournaments.WC2026Structure)
		rec.Set("sync", tournaments.WC2026Sync)
		rec.Set("extIdPrefix", "WC2026")
		// Tournament window = the seeded fixture range when present (an
		// upgraded live DB), else the official dates (fresh DB, pre-seed).
		startsAt, endsAt := "2026-06-11 19:00:00.000Z", "2026-07-19 19:00:00.000Z"
		if first, err := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 1, 0); err == nil && len(first) == 1 {
			startsAt = first[0].GetString("kickoff")
		}
		if last, err := app.FindRecordsByFilter("matches", "id != ''", "-kickoff", 1, 0); err == nil && len(last) == 1 {
			endsAt = last[0].GetString("kickoff")
		}
		rec.Set("startsAt", startsAt)
		rec.Set("endsAt", endsAt)
		if def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true"); err == nil {
			rec.Set("scoringConfig", def.Id)
		}
		return app.Save(rec)
	}, func(app core.App) error {
		if col, err := app.FindCollectionByNameOrId(nTournaments); err == nil {
			return app.Delete(col)
		}
		return nil
	})
}
