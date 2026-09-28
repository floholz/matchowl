package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Mailings and richer announcements (2026-09-28, admin app step 4).
//
// `mailings`: one targeted email to an audience — subject, Markdown body,
// a call-to-action button, the audience filter as stored JSON, and what
// happened when it was sent. Delivery goes through the notifications
// ledger (event "mailing", dedup key "mailing:<id>"), so a second send is
// a no-op. Announcements gain the same call-to-action fields.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if _, err := app.FindCollectionByNameOrId("mailings"); err != nil {
			c := core.NewBaseCollection("mailings")
			admin := "@request.auth.role = 'admin' || @request.auth.role = 'owner'"
			c.ListRule, c.ViewRule = &admin, &admin
			c.Fields.Add(&core.TextField{Name: "subject", Required: true, Max: 200})
			c.Fields.Add(&core.TextField{Name: "body", Required: true, Max: 20000})
			c.Fields.Add(&core.TextField{Name: "ctaText", Max: 60})
			c.Fields.Add(&core.TextField{Name: "ctaUrl", Max: 500})
			c.Fields.Add(&core.JSONField{Name: "audience", MaxSize: 8000})
			c.Fields.Add(&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"draft", "sent"}})
			c.Fields.Add(&core.DateField{Name: "sentAt"})
			c.Fields.Add(&core.RelationField{Name: "sentBy", CollectionId: users.Id, MaxSelect: 1})
			c.Fields.Add(&core.NumberField{Name: "recipients", OnlyInt: true})
			c.Fields.Add(&core.JSONField{Name: "result", MaxSize: 2000})
			c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
			if err := app.Save(c); err != nil {
				return err
			}
		}
		a, err := app.FindCollectionByNameOrId("announcements")
		if err != nil {
			return err
		}
		if a.Fields.GetByName("ctaText") == nil {
			a.Fields.Add(&core.TextField{Name: "ctaText", Max: 60})
		}
		if a.Fields.GetByName("ctaUrl") == nil {
			a.Fields.Add(&core.TextField{Name: "ctaUrl", Max: 500})
		}
		return app.Save(a)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("mailings"); err == nil {
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		a, err := app.FindCollectionByNameOrId("announcements")
		if err != nil {
			return err
		}
		a.Fields.RemoveByName("ctaText")
		a.Fields.RemoveByName("ctaUrl")
		return app.Save(a)
	})
}
