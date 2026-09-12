package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// pool_invites: a member invites a friend into an open pool; the invitee
// accepts (joins) or declines (row deleted). Written only by /api/pools.
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("pool_invites"); err == nil {
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
		c := core.NewBaseCollection("pool_invites")
		c.Fields.Add(&core.RelationField{Name: "pool", CollectionId: pools.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		c.Fields.Add(&core.RelationField{Name: "inviter", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		c.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.AddIndex("idx_pool_invites_pair", true, "pool, user", "")
		rule := "user = @request.auth.id || inviter = @request.auth.id"
		c.ListRule = &rule
		c.ViewRule = &rule
		return app.Save(c)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("pool_invites"); err == nil {
			return app.Delete(c)
		}
		return nil
	})
}
