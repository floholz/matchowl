package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// users.pinnedCompetitions (2026-11-23, PLAN.md "pools at the center"): the
// competition keys a user pinned to the Matches filter row, in order. The
// competitions their pools play are pinned anyway and aren't stored here.
// Kept on the account so the pins follow you across devices; written by the
// user (frontend/src/lib/pins.svelte.ts).
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("pinnedCompetitions") != nil {
			return nil
		}
		c.Fields.Add(&core.JSONField{Name: "pinnedCompetitions", MaxSize: 2000})
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}
		c.Fields.RemoveByName("pinnedCompetitions")
		return app.Save(c)
	})
}
