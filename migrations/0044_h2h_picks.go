package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// h2h_picks (2026-09-14, head-to-head pools): a member's save calls
// (matches that count double for them) and ban (a match taken away from
// their rival) per pool and matchday. Placed and moved until the match
// kicks off, hidden from everyone else until then. Server-only, no API
// rules — see /api/pools/{id}/h2h/picks.
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("h2h_picks"); err == nil {
			return nil
		}
		pools, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		matches, err := app.FindCollectionByNameOrId("matches")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("h2h_picks")
		c.Fields.Add(&core.RelationField{Name: "pool", CollectionId: pools.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		c.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		c.Fields.Add(&core.RelationField{Name: "match", CollectionId: matches.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		c.Fields.Add(&core.TextField{Name: "round", Required: true, Max: 80}) // the match's round key
		c.Fields.Add(&core.SelectField{Name: "kind", Values: []string{"save", "ban"}, MaxSelect: 1, Required: true})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		c.AddIndex("idx_h2h_picks_unique", true, "pool, user, match, kind", "")
		c.AddIndex("idx_h2h_picks_round", false, "pool, round", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("h2h_picks"); err == nil {
			return app.Delete(c)
		}
		return nil
	})
}
