package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Live data, phase 4 of "pools at the center" (2026-11-24, PLAN.md):
//
//   - `match_events`: the ticker of a match (goals, cards, substitutions,
//     VAR), written by the sync from the provider's live and by-id answers.
//     The provider gives events no id, so `key` is derived from the event's
//     content plus its occurrence (package sync); rows are upserted on it and
//     dropped when the provider drops them (a VAR-cancelled goal). Public
//     like matches; realtime pushes them to the match page.
//   - `matches.afId`: the API-Football fixture id, so the live poller can
//     map answers to rows and check just-finished matches by id in batches.
func init() {
	m.Register(func(app core.App) error {
		matches, err := app.FindCollectionByNameOrId("matches")
		if err != nil {
			return err
		}
		if matches.Fields.GetByName("afId") == nil {
			matches.Fields.Add(&core.NumberField{Name: "afId", OnlyInt: true})
			matches.AddIndex("idx_matches_af_id", false, "afId", "")
			if err := app.Save(matches); err != nil {
				return err
			}
		}
		if _, err := app.FindCollectionByNameOrId("match_events"); err == nil {
			return nil
		}
		teams, err := app.FindCollectionByNameOrId("teams")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("match_events")
		c.ListRule = ptr("")
		c.ViewRule = ptr("")
		c.Fields.Add(&core.RelationField{Name: "match", Required: true, CollectionId: matches.Id, MaxSelect: 1, CascadeDelete: true})
		c.Fields.Add(&core.TextField{Name: "key", Required: true, Max: 200})
		c.Fields.Add(&core.NumberField{Name: "seq", OnlyInt: true})
		c.Fields.Add(&core.NumberField{Name: "minute", OnlyInt: true})
		c.Fields.Add(&core.NumberField{Name: "extra", OnlyInt: true})
		c.Fields.Add(&core.TextField{Name: "type", Max: 32})
		c.Fields.Add(&core.TextField{Name: "detail", Max: 64})
		c.Fields.Add(&core.SelectField{Name: "side", MaxSelect: 1, Values: []string{"home", "away"}})
		c.Fields.Add(&core.RelationField{Name: "team", CollectionId: teams.Id, MaxSelect: 1})
		c.Fields.Add(&core.TextField{Name: "player", Max: 100})
		c.Fields.Add(&core.TextField{Name: "assist", Max: 100})
		c.Fields.Add(&core.TextField{Name: "comments", Max: 200})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		c.AddIndex("idx_match_events_match_key", true, "match, key", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("match_events"); err == nil {
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		if matches, err := app.FindCollectionByNameOrId("matches"); err == nil {
			matches.RemoveIndex("idx_matches_af_id")
			matches.Fields.RemoveByName("afId")
			return app.Save(matches)
		}
		return nil
	})
}
