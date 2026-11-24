package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Challenges, phase 3 of "pools at the center" (2026-11-24, PLAN.md): one
// friend challenges another on one match. The opponent accepts before
// kick-off, both tip as usual, and the default scoring decides (more points
// wins, equal is a draw). `stake` is a free-text note the app only shows.
// The points are written when the match is final (package challenges), so
// the record per friend is a count, not a recompute.
//
// Rows are only written through /api/challenges, so the collection has no
// API rules (superusers only).
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("challenges"); err == nil {
			return nil
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		matches, err := app.FindCollectionByNameOrId("matches")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("challenges")
		c.Fields.Add(&core.RelationField{Name: "match", Required: true, CollectionId: matches.Id, MaxSelect: 1, CascadeDelete: true})
		c.Fields.Add(&core.RelationField{Name: "challenger", Required: true, CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true})
		c.Fields.Add(&core.RelationField{Name: "opponent", Required: true, CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true})
		c.Fields.Add(&core.TextField{Name: "stake", Max: 140})
		c.Fields.Add(&core.SelectField{Name: "status", Required: true, MaxSelect: 1,
			Values: []string{"pending", "accepted", "declined", "cancelled"}})
		c.Fields.Add(&core.NumberField{Name: "challengerPts", OnlyInt: true})
		c.Fields.Add(&core.NumberField{Name: "opponentPts", OnlyInt: true})
		c.Fields.Add(&core.DateField{Name: "settledAt"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		c.AddIndex("idx_challenges_match", false, "match", "")
		c.AddIndex("idx_challenges_challenger", false, "challenger", "")
		c.AddIndex("idx_challenges_opponent", false, "opponent", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("challenges"); err == nil {
			return app.Delete(c)
		}
		return nil
	})
}
