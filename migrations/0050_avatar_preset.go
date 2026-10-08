package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// users.avatarPreset (2026-10-08): the drawn profile picture a user picked,
// as "<template>:<palette>" (e.g. "raven:ember"). Drawn by the client
// (frontend/src/lib/avatars.ts), never stored as an image. An uploaded
// photo (`avatar`) wins; empty = a default palette from the user id.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("avatarPreset") != nil {
			return nil
		}
		c.Fields.Add(&core.TextField{Name: "avatarPreset", Max: 40, Pattern: `^([a-z]+:[a-z]+)?$`})
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return nil
		}
		c.Fields.RemoveByName("avatarPreset")
		return app.Save(c)
	})
}
