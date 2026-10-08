package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// signup_links.pools (2026-10-08): the pools a link puts its account into
// (the alpha / beta tester pools). Joined once the account is verified —
// the same moment it joins Global — by internal/pools.
func init() {
	m.Register(func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("signup_links")
		if err != nil {
			return err
		}
		if c.Fields.GetByName("pools") != nil {
			return nil
		}
		pools, err := app.FindCollectionByNameOrId("pools")
		if err != nil {
			return err
		}
		c.Fields.Add(&core.RelationField{Name: "pools", CollectionId: pools.Id, MaxSelect: 20})
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("signup_links")
		if err != nil {
			return nil
		}
		c.Fields.RemoveByName("pools")
		return app.Save(c)
	})
}
