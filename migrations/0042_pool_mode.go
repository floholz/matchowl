package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/floholz/matchowl/internal/tournaments"
)

// Pool mode (2026-09-14, head-to-head pools): a pool binds exactly one
// season (`tournaments` drops from up to 20 to 1) and plays it in a mode —
// `classic` (the points table) or `h2h` (matchday duels). `saveCalls` is
// the h2h allowance of matches that count double per matchday (1 or 2).
// Existing pools keep their running season (else the first) and become
// classic, which is what they were.
func init() {
	m.Register(func(app core.App) error {
		pools, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}
		if pools.Fields.GetByName("mode") == nil {
			pools.Fields.Add(&core.SelectField{Name: "mode", Values: []string{"classic", "h2h"}, MaxSelect: 1})
		}
		if pools.Fields.GetByName("saveCalls") == nil {
			pools.Fields.Add(&core.NumberField{Name: "saveCalls", OnlyInt: true})
		}
		if f, ok := pools.Fields.GetByName("tournaments").(*core.RelationField); ok {
			f.MaxSelect = 1
		}
		if err := app.Save(pools); err != nil {
			return err
		}
		recs, err := app.FindRecordsByFilter("pools", "id != ''", "", 0, 0)
		if err != nil {
			return err
		}
		for _, lg := range recs {
			ts := lg.GetStringSlice("tournaments")
			if len(ts) > 1 {
				keep := ts[0]
				for _, id := range ts {
					if t, err := app.FindRecordById("tournaments", id); err == nil && t.GetString("status") == tournaments.StatusActive {
						keep = id
						break
					}
				}
				lg.Set("tournaments", []string{keep})
			}
			if lg.GetString("mode") == "" {
				lg.Set("mode", "classic")
			}
			if lg.GetInt("saveCalls") == 0 {
				lg.Set("saveCalls", 1)
			}
			if err := app.Save(lg); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		pools, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}
		pools.Fields.RemoveByName("mode")
		pools.Fields.RemoveByName("saveCalls")
		if f, ok := pools.Fields.GetByName("tournaments").(*core.RelationField); ok {
			f.MaxSelect = 20
		}
		return app.Save(pools)
	})
}
