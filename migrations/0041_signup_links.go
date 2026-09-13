package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// signup_links: one-time registration links for the closed test (2026-09-13).
// An admin mints a link from the Admin area; whoever opens it may create one
// account while REGISTRATION_OPEN=0 (email/password or Google). The row
// records who it was for (label), who made it, and who used it.
//
// No API rules: the collection is only reachable through the admin endpoints
// in internal/users (plus the anonymous token check the register page uses).
func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId("signup_links"); err == nil {
			return nil
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("signup_links")
		c.Fields.Add(&core.TextField{Name: "token", Required: true, Min: 16, Max: 64})
		// Free-text note: who the link is for ("Anna", "the office group").
		c.Fields.Add(&core.TextField{Name: "label", Max: 80})
		c.Fields.Add(&core.RelationField{Name: "createdBy", CollectionId: users.Id, MaxSelect: 1})
		c.Fields.Add(&core.RelationField{Name: "usedBy", CollectionId: users.Id, MaxSelect: 1})
		c.Fields.Add(&core.DateField{Name: "usedAt"})
		// Empty = never expires.
		c.Fields.Add(&core.DateField{Name: "expiresAt"})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		c.AddIndex("idx_signup_links_token", true, "token", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("signup_links"); err == nil {
			return app.Delete(c)
		}
		return nil
	})
}
