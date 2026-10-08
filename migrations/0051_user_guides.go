package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// users.guides (2026-10-08): which in-app guides a user has been through,
// as {"<guide id>": "<when>"}. Kept on the account rather than the device
// so the PWA and the desktop don't each replay them. Written by the user
// (frontend/src/lib/guide.svelte.ts); only they can read it (users rules).
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("guides") != nil {
			return nil
		}
		c.Fields.Add(&core.JSONField{Name: "guides", MaxSize: 4000})
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}
		c.Fields.RemoveByName("guides")
		return app.Save(c)
	})
}
