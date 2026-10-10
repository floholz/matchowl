package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Match clock (2026-10-10): what the provider said about a live match's
// minute at the last sync — the phase (1H, HT, 2H, ET, BT, P), the minute,
// stoppage minutes past 45/90/120, and when that was read (liveAt). The
// client counts on from liveAt between syncs. All empty unless live; a
// clock-only change never triggers a score recompute (scoring.Register).
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("matches")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("livePhase") == nil {
			c.Fields.Add(&core.TextField{Name: "livePhase", Max: 8})
		}
		if c.Fields.GetByName("liveMinute") == nil {
			c.Fields.Add(&core.NumberField{Name: "liveMinute", OnlyInt: true})
		}
		if c.Fields.GetByName("liveExtra") == nil {
			c.Fields.Add(&core.NumberField{Name: "liveExtra", OnlyInt: true})
		}
		if c.Fields.GetByName("liveAt") == nil {
			c.Fields.Add(&core.DateField{Name: "liveAt"})
		}
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("matches")
		if err != nil {
			return nil
		}
		for _, n := range []string{"livePhase", "liveMinute", "liveExtra", "liveAt"} {
			c.Fields.RemoveByName(n)
		}
		return app.Save(c)
	})
}
