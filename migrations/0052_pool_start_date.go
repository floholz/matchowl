package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// pools.startDate (2026-10-10): a pool's standings can restart part-way
// through its season — only matches kicking off at or after it count, the
// Forecast drops out when it falls after the season's start, and a
// head-to-head table counts only the rounds that kicked off from then on.
// Empty = the whole season, which is what every pool counted so far. Set
// by the owner (POST /api/pools/{id}/start).
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("startDate") != nil {
			return nil
		}
		c.Fields.Add(&core.DateField{Name: "startDate"})
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return nil
		}
		c.Fields.RemoveByName("startDate")
		return app.Save(c)
	})
}
