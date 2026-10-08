package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// pools.h2hNext (2026-10-08): the pairings of a head-to-head pool's next
// round, kept once the roster changes so a join or leave patches them
// (Ghost in, Ghost out) instead of re-rotating everyone. Hidden: served
// through /api/pools/{id}/h2h only. See internal/h2h/next.go.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("h2hNext") != nil {
			return nil
		}
		c.Fields.Add(&core.JSONField{Name: "h2hNext", MaxSize: 20000, Hidden: true})
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return nil
		}
		c.Fields.RemoveByName("h2hNext")
		return app.Save(c)
	})
}
