package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Provider ids on teams (2026-09-28, admin app step 3): which data source
// a team row came from and its id there (API-Football today), so the
// teams registry can cross-reference one club or nation across seasons
// and the importer can match by id instead of by name. Written at import;
// older rows are linked by the crest refresh (name match).
func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("teams")
		if err != nil {
			return err
		}
		if col.Fields.GetByName("provider") == nil {
			col.Fields.Add(&core.TextField{Name: "provider", Max: 32})
		}
		if col.Fields.GetByName("providerId") == nil {
			col.Fields.Add(&core.NumberField{Name: "providerId", OnlyInt: true})
		}
		col.AddIndex("idx_teams_provider_id", false, "provider, providerId", "")
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("teams")
		if err != nil {
			return err
		}
		col.RemoveIndex("idx_teams_provider_id")
		col.Fields.RemoveByName("provider")
		col.Fields.RemoveByName("providerId")
		return app.Save(col)
	})
}
