package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// System chat messages (2026-09-14, head-to-head pools): the app posts a
// matchday's duel results into the pool chat. Such a message has no user
// and `system = true`; the chat renders it as a note and the chat notifier
// skips it (the round result has its own notification).
func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("pool_messages")
		if err != nil {
			return err
		}
		if f, ok := col.Fields.GetByName("user").(*core.RelationField); ok {
			f.Required = false
		}
		if col.Fields.GetByName("system") == nil {
			col.Fields.Add(&core.BoolField{Name: "system"})
		}
		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("pool_messages")
		if err != nil {
			return err
		}
		col.Fields.RemoveByName("system")
		if f, ok := col.Fields.GetByName("user").(*core.RelationField); ok {
			f.Required = true
		}
		return app.Save(col)
	})
}
