package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Pools at the center (2026-11-23, PLAN.md): nobody "plays" a competition
// any more. "My matches" comes from pools, followed teams, challenges and
// tipped matches (package scope), so tournament participation goes away.
// Down does not bring the rows back.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("tournament_players")
		if err != nil {
			return nil // already gone
		}
		return app.Delete(c)
	}, func(app core.App) error {
		return nil
	})
}
