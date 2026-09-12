package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Friends + pools (PLAN.md §7 verdict, 2026-09-13):
//   - leagues (the pools) get `tournaments`: the seasons a pool counts. Every
//     existing private league was a WC 2026 league, so it is bound to that
//     season; the Global league stays unbound (the everyone board).
//   - friendships: a mutual social graph. One side requests, the other
//     accepts. Rows are only ever written by the /api/friends endpoints.
func init() {
	m.Register(func(app core.App) error {
		tCol, err := app.FindCollectionByNameOrId("tournaments")
		if err != nil {
			return err
		}
		leagues, err := app.FindCollectionByNameOrId("leagues")
		if err != nil {
			return err
		}
		if leagues.Fields.GetByName("tournaments") == nil {
			leagues.Fields.Add(&core.RelationField{Name: "tournaments", CollectionId: tCol.Id, MaxSelect: 20})
			if err := app.Save(leagues); err != nil {
				return err
			}
		}
		if wc, err := app.FindFirstRecordByFilter("tournaments", "slug = 'wc2026'"); err == nil {
			recs, err := app.FindRecordsByFilter("leagues", "inviteCode != 'GLOBAL'", "", 0, 0)
			if err != nil {
				return err
			}
			for _, r := range recs {
				if len(r.GetStringSlice("tournaments")) == 0 {
					r.Set("tournaments", []string{wc.Id})
					if err := app.Save(r); err != nil {
						return err
					}
				}
			}
		}

		if _, err := app.FindCollectionByNameOrId("friendships"); err == nil {
			return nil
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		fr := core.NewBaseCollection("friendships")
		fr.Fields.Add(&core.RelationField{Name: "requester", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		fr.Fields.Add(&core.RelationField{Name: "receiver", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		fr.Fields.Add(&core.SelectField{Name: "status", Values: []string{"pending", "accepted"}, MaxSelect: 1, Required: true})
		fr.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		fr.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		fr.AddIndex("idx_friendships_pair", true, "requester, receiver", "")
		// Members see their own edges; every write goes through the API.
		rule := "requester = @request.auth.id || receiver = @request.auth.id"
		fr.ListRule = &rule
		fr.ViewRule = &rule
		return app.Save(fr)
	}, func(app core.App) error {
		if fr, err := app.FindCollectionByNameOrId("friendships"); err == nil {
			if err := app.Delete(fr); err != nil {
				return err
			}
		}
		leagues, err := app.FindCollectionByNameOrId("leagues")
		if err != nil {
			return err
		}
		leagues.Fields.RemoveByName("tournaments")
		return app.Save(leagues)
	})
}
