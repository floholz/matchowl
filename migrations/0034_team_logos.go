package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Admin tournament browser (PLAN-admin-tournaments.md): club crests.
//
//   - teams.logo (file, ≤1 MB image): downloaded from the results provider
//     at import time (or via the admin "fetch logos" action) and served
//     from our own /api/files, so players never load third-party images.
//     National teams keep using the bundled ISO flags; the crest is the
//     fallback when no flag is bundled.
func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId(nTeams)
		if err != nil {
			return err
		}
		if col.Fields.GetByName("logo") != nil {
			return nil
		}
		col.Fields.Add(&core.FileField{
			Name: "logo", MaxSelect: 1, MaxSize: 1 << 20,
			MimeTypes: []string{"image/png", "image/svg+xml", "image/webp", "image/jpeg", "image/gif"},
		})
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId(nTeams)
		if err != nil {
			return err
		}
		if col.Fields.GetByName("logo") == nil {
			return nil
		}
		col.Fields.RemoveByName("logo")
		return app.Save(col)
	})
}
