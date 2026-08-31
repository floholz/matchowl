package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The UEFA Swiss-model league phase (UCL/UEL/UECL) has 36 teams in one
// table: tournament_groups.teams MaxSelect 24 → 36 (the structure groupSize
// cap is raised alongside in Go).
func init() {
	m.Register(func(app core.App) error {
		gcol, err := app.FindCollectionByNameOrId("tournament_groups")
		if err != nil {
			return err
		}
		if f, ok := gcol.Fields.GetByName("teams").(*core.RelationField); ok && f.MaxSelect < 36 {
			f.MaxSelect = 36
			return app.Save(gcol)
		}
		return nil
	}, func(app core.App) error {
		gcol, err := app.FindCollectionByNameOrId("tournament_groups")
		if err != nil {
			return err
		}
		if f, ok := gcol.Fields.GetByName("teams").(*core.RelationField); ok {
			f.MaxSelect = 24
			return app.Save(gcol)
		}
		return nil
	})
}
