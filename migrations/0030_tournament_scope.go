package migrations

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Multi-tournament rework, step 2: scope the tournament-shaped collections to
// the tournaments entity. Adds a required `tournament` relation to teams,
// tournament_groups, matches, forecasts and forecast_scores (tips and
// match_scores derive theirs via `match`), backfills every existing row to
// wc2026, swaps the uniqueness indexes that encoded "there is only one
// tournament", and relaxes matches.stage from a fixed-values Select to Text
// (stage vocabulary is validated in Go against the tournament's structure).
//
// On a fresh DB this runs before the seeder and only performs the schema part.
func init() {
	m.Register(up0030, down0030)
}

// scopeSpec describes the per-collection changes: the index that encoded
// single-tournament uniqueness (dropped) and its tournament-scoped
// replacement.
type scopeSpec struct {
	col        string
	dropIdx    string
	addIdx     string
	addIdxCols string
	addUnique  bool
}

var scopeSpecs = []scopeSpec{
	{"teams", "idx_teams_fifaCode", "idx_teams_tournament_fifaCode", "tournament, fifaCode", true},
	{"tournament_groups", "idx_groups_letter", "idx_groups_tournament_letter", "tournament, letter", true},
	{"matches", "", "idx_matches_tournament", "tournament", false}, // extId stays globally unique (prefix-scoped)
	{"forecasts", "idx_forecasts_user", "idx_forecasts_user_tournament", "user, tournament", true},
	{"forecast_scores", "idx_fs_user_config", "idx_fs_user_tournament_config", "user, tournament, config", true},
}

func up0030(app core.App) error {
	t, err := app.FindFirstRecordByFilter(nTournaments, "slug = 'wc2026'")
	if err != nil {
		return fmt.Errorf("wc2026 tournament record missing (0029 must run first): %w", err)
	}
	tCol, err := app.FindCollectionByNameOrId(nTournaments)
	if err != nil {
		return err
	}

	for _, s := range scopeSpecs {
		col, err := app.FindCollectionByNameOrId(s.col)
		if err != nil {
			return err
		}
		if col.Fields.GetByName("tournament") != nil {
			continue // already migrated
		}
		// 1. Add as optional so existing rows stay valid, then backfill.
		col.Fields.Add(&core.RelationField{
			Name: "tournament", CollectionId: tCol.Id, MaxSelect: 1, CascadeDelete: true,
		})
		if err := app.Save(col); err != nil {
			return fmt.Errorf("%s: add tournament field: %w", s.col, err)
		}
		if _, err := app.DB().NewQuery(
			"UPDATE {{" + s.col + "}} SET [[tournament]] = {:t} WHERE [[tournament]] = ''",
		).Bind(dbx.Params{"t": t.Id}).Execute(); err != nil {
			return fmt.Errorf("%s: backfill: %w", s.col, err)
		}
		// 2. Require it and swap the single-tournament uniqueness index.
		col, err = app.FindCollectionByNameOrId(s.col)
		if err != nil {
			return err
		}
		if f, ok := col.Fields.GetByName("tournament").(*core.RelationField); ok {
			f.Required = true
		}
		if s.dropIdx != "" {
			col.RemoveIndex(s.dropIdx)
		}
		col.AddIndex(s.addIdx, s.addUnique, s.addIdxCols, "")
		if err := app.Save(col); err != nil {
			return fmt.Errorf("%s: require + reindex: %w", s.col, err)
		}
	}

	return relaxStageField(app)
}

// relaxStageField converts matches.stage from a Select with the hardcoded
// WC2026 values to a plain Text field, preserving data via a temp column
// (PocketBase cannot change a field's type in place).
func relaxStageField(app core.App) error {
	col, err := app.FindCollectionByNameOrId("matches")
	if err != nil {
		return err
	}
	if _, isSelect := col.Fields.GetByName("stage").(*core.SelectField); !isSelect {
		return nil // already text (or gone) — nothing to do
	}

	col.Fields.Add(&core.TextField{Name: "stageTmp", Max: 16})
	if err := app.Save(col); err != nil {
		return err
	}
	if _, err := app.DB().NewQuery("UPDATE {{matches}} SET [[stageTmp]] = [[stage]]").Execute(); err != nil {
		return err
	}

	col, err = app.FindCollectionByNameOrId("matches")
	if err != nil {
		return err
	}
	col.Fields.RemoveByName("stage")
	if err := app.Save(col); err != nil {
		return err
	}

	col, err = app.FindCollectionByNameOrId("matches")
	if err != nil {
		return err
	}
	col.Fields.Add(&core.TextField{Name: "stage", Required: true, Max: 16})
	if err := app.Save(col); err != nil {
		return err
	}
	if _, err := app.DB().NewQuery("UPDATE {{matches}} SET [[stage]] = [[stageTmp]]").Execute(); err != nil {
		return err
	}

	col, err = app.FindCollectionByNameOrId("matches")
	if err != nil {
		return err
	}
	col.Fields.RemoveByName("stageTmp")
	return app.Save(col)
}

func down0030(app core.App) error {
	// Restore the Select stage field (data preserved via the same dance).
	col, err := app.FindCollectionByNameOrId("matches")
	if err == nil {
		if _, isText := col.Fields.GetByName("stage").(*core.TextField); isText {
			col.Fields.Add(&core.TextField{Name: "stageTmp", Max: 16})
			if err := app.Save(col); err != nil {
				return err
			}
			if _, err := app.DB().NewQuery("UPDATE {{matches}} SET [[stageTmp]] = [[stage]]").Execute(); err != nil {
				return err
			}
			col, _ = app.FindCollectionByNameOrId("matches")
			col.Fields.RemoveByName("stage")
			if err := app.Save(col); err != nil {
				return err
			}
			col, _ = app.FindCollectionByNameOrId("matches")
			col.Fields.Add(&core.SelectField{
				Name: "stage", Required: true, MaxSelect: 1,
				Values: []string{"group", "R32", "R16", "QF", "SF", "3RD", "FINAL"},
			})
			if err := app.Save(col); err != nil {
				return err
			}
			if _, err := app.DB().NewQuery("UPDATE {{matches}} SET [[stage]] = [[stageTmp]]").Execute(); err != nil {
				return err
			}
			col, _ = app.FindCollectionByNameOrId("matches")
			col.Fields.RemoveByName("stageTmp")
			if err := app.Save(col); err != nil {
				return err
			}
		}
	}

	for _, s := range scopeSpecs {
		col, err := app.FindCollectionByNameOrId(s.col)
		if err != nil {
			continue
		}
		if col.Fields.GetByName("tournament") == nil {
			continue
		}
		col.Fields.RemoveByName("tournament")
		col.RemoveIndex(s.addIdx)
		switch s.col {
		case "teams":
			col.AddIndex("idx_teams_fifaCode", true, "fifaCode", "")
		case "tournament_groups":
			col.AddIndex("idx_groups_letter", true, "letter", "")
		case "forecasts":
			col.AddIndex("idx_forecasts_user", true, "user", "")
		case "forecast_scores":
			col.AddIndex("idx_fs_user_config", true, "user, config", "")
		}
		if err := app.Save(col); err != nil {
			return err
		}
	}
	return nil
}
