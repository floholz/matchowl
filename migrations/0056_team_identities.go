package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Teams, phase 2 of "pools at the center" (2026-11-24, PLAN.md):
//
//   - `team_identities`: one row per club or nation, across seasons. Team
//     pages (/teams/{key}), following and search hang off it. Each season's
//     `teams` row points to it through `teams.identity`; rows are linked by
//     package teams (on create, and a sweep at boot that backfills existing
//     rows from the provider id, the club key or the flag code).
//     `country` is the ISO code of the team's country (the nation itself
//     for national teams), from the API's /teams answer.
//   - `team_follows`: who follows which team. Followed teams' matches are
//     part of "my matches" (package scope).
//   - `users.country`: the user's country (ISO code) for the country chip on
//     Matches.
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("team_identities"); err == nil {
			return nil
		}
		ids := core.NewBaseCollection("team_identities")
		ids.ListRule = ptr("")
		ids.ViewRule = ptr("")
		ids.Fields.Add(&core.TextField{Name: "key", Required: true, Max: 64})
		ids.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 100})
		ids.Fields.Add(&core.TextField{Name: "fifaCode", Max: 3})
		ids.Fields.Add(&core.TextField{Name: "iso2", Max: 8})
		ids.Fields.Add(&core.BoolField{Name: "national"})
		ids.Fields.Add(&core.TextField{Name: "country", Max: 8})
		ids.Fields.Add(&core.TextField{Name: "countryName", Max: 64})
		ids.Fields.Add(&core.TextField{Name: "provider", Max: 32})
		ids.Fields.Add(&core.NumberField{Name: "providerId", OnlyInt: true})
		ids.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		ids.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		ids.AddIndex("idx_team_identities_key", true, "key", "")
		ids.AddIndex("idx_team_identities_provider", false, "provider, providerId", "")
		ids.AddIndex("idx_team_identities_country", false, "country", "")
		if err := app.Save(ids); err != nil {
			return err
		}

		teams, err := app.FindCollectionByNameOrId("teams")
		if err != nil {
			return err
		}
		teams.Fields.Add(&core.RelationField{Name: "identity", CollectionId: ids.Id, MaxSelect: 1})
		teams.AddIndex("idx_teams_identity", false, "identity", "")
		if err := app.Save(teams); err != nil {
			return err
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		follows := core.NewBaseCollection("team_follows")
		own := "@request.auth.id != '' && user = @request.auth.id"
		follows.ListRule = ptr(own)
		follows.ViewRule = ptr(own)
		follows.CreateRule = ptr("@request.auth.id != '' && @request.body.user = @request.auth.id")
		follows.DeleteRule = ptr(own)
		follows.Fields.Add(&core.RelationField{Name: "user", Required: true, CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true})
		follows.Fields.Add(&core.RelationField{Name: "team", Required: true, CollectionId: ids.Id, MaxSelect: 1, CascadeDelete: true})
		follows.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		follows.AddIndex("idx_team_follows_user_team", true, "user, team", "")
		follows.AddIndex("idx_team_follows_team", false, "team", "")
		if err := app.Save(follows); err != nil {
			return err
		}

		if users.Fields.GetByName("country") == nil {
			users.Fields.Add(&core.TextField{Name: "country", Max: 8})
			return app.Save(users)
		}
		return nil
	}, func(app core.App) error {
		if users, err := app.FindCollectionByNameOrId("users"); err == nil {
			users.Fields.RemoveByName("country")
			if err := app.Save(users); err != nil {
				return err
			}
		}
		if c, err := app.FindCollectionByNameOrId("team_follows"); err == nil {
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		if teams, err := app.FindCollectionByNameOrId("teams"); err == nil {
			teams.RemoveIndex("idx_teams_identity")
			teams.Fields.RemoveByName("identity")
			if err := app.Save(teams); err != nil {
				return err
			}
		}
		if c, err := app.FindCollectionByNameOrId("team_identities"); err == nil {
			return app.Delete(c)
		}
		return nil
	})
}
