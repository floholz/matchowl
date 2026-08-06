package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Feed rework, step 4 (PLAN-feed.md): the forecast's shape becomes
// per-tournament admin config.
//
//   - tournaments.forecastSpec (JSON): {"mode":"full"} (the ceremonial
//     builder, wc2026 backfill), {"mode":"calls","calls":[...]} (headline
//     calls — champion / UEFA spots / relegation — for league seasons), or
//     {"mode":"none"}. Missing spec = full, so pre-spec records keep the
//     old behavior.
//   - forecasts.calls (JSON): {callKey: teamId | [teamIds]} for calls mode.
//   - tournament_groups.teams MaxSelect 8 → 24 so a league season fits in
//     one group (structure groupSize cap is raised alongside in Go).
func init() {
	m.Register(func(app core.App) error {
		tcol, err := app.FindCollectionByNameOrId(nTournaments)
		if err != nil {
			return err
		}
		if tcol.Fields.GetByName("forecastSpec") != nil {
			return nil // already migrated
		}
		tcol.Fields.Add(&core.JSONField{Name: "forecastSpec", MaxSize: 8000})
		if err := app.Save(tcol); err != nil {
			return err
		}
		if t, err := app.FindFirstRecordByFilter(nTournaments, "slug = 'wc2026'"); err == nil {
			t.Set("forecastSpec", `{"mode":"full"}`)
			if err := app.Save(t); err != nil {
				return err
			}
		}

		fcol, err := app.FindCollectionByNameOrId("forecasts")
		if err != nil {
			return err
		}
		fcol.Fields.Add(&core.JSONField{Name: "calls", MaxSize: 8000})
		if err := app.Save(fcol); err != nil {
			return err
		}

		gcol, err := app.FindCollectionByNameOrId("tournament_groups")
		if err != nil {
			return err
		}
		if f, ok := gcol.Fields.GetByName("teams").(*core.RelationField); ok && f.MaxSelect < 24 {
			f.MaxSelect = 24
			if err := app.Save(gcol); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		if tcol, err := app.FindCollectionByNameOrId(nTournaments); err == nil {
			tcol.Fields.RemoveByName("forecastSpec")
			if err := app.Save(tcol); err != nil {
				return err
			}
		}
		if fcol, err := app.FindCollectionByNameOrId("forecasts"); err == nil {
			fcol.Fields.RemoveByName("calls")
			if err := app.Save(fcol); err != nil {
				return err
			}
		}
		if gcol, err := app.FindCollectionByNameOrId("tournament_groups"); err == nil {
			if f, ok := gcol.Fields.GetByName("teams").(*core.RelationField); ok {
				f.MaxSelect = 8
				return app.Save(gcol)
			}
		}
		return nil
	})
}
