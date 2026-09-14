package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// h2h_rounds (2026-09-14, head-to-head pools): one row per pool and
// matchday the pool played. Written by the h2h job: opened at the round's
// first kick-off with the pairings frozen from the roster of that moment,
// closed 24 h after its last scheduled kick-off with the results frozen.
// History never changes afterwards. No API rules — served by
// /api/pools/{id}/h2h only.
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("h2h_rounds"); err == nil {
			return nil
		}
		pools, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("h2h_rounds")
		c.Fields.Add(&core.RelationField{Name: "pool", CollectionId: pools.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		// The matchday: `stage|roundLabel`, the key the hub filters on.
		c.Fields.Add(&core.TextField{Name: "key", Required: true, Max: 80})
		c.Fields.Add(&core.TextField{Name: "stage", Max: 16})
		c.Fields.Add(&core.TextField{Name: "label", Max: 64})
		c.Fields.Add(&core.NumberField{Name: "num", OnlyInt: true})
		// 0-based count of rounds the pool had played when this one opened;
		// drives the pairing rotation.
		c.Fields.Add(&core.NumberField{Name: "ordinal", OnlyInt: true})
		c.Fields.Add(&core.DateField{Name: "firstKickoff"})
		c.Fields.Add(&core.DateField{Name: "closesAt"})
		c.Fields.Add(&core.SelectField{Name: "status", Values: []string{"open", "closed"}, MaxSelect: 1, Required: true})
		c.Fields.Add(&core.JSONField{Name: "pairings", MaxSize: 20000}) // [{a, b}] — b "" = the Ghost
		c.Fields.Add(&core.JSONField{Name: "matches", MaxSize: 20000})  // match ids of the round at open time
		c.Fields.Add(&core.JSONField{Name: "results", MaxSize: 60000})  // frozen at close (see h2h.Results)
		c.Fields.Add(&core.DateField{Name: "closedAt"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		c.AddIndex("idx_h2h_rounds_pool_key", true, "pool, key", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("h2h_rounds"); err == nil {
			return app.Delete(c)
		}
		return nil
	})
}
