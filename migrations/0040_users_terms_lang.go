package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Legal acceptance + language on users (2026-09-13).
//
//   - termsAcceptedAt / termsVersion: when and which version of the terms +
//     privacy notice the person accepted. The register form sends them with
//     the create call; Google sign-ups (and any account whose version is
//     stale) get an accept interstitial on their next visit — the frontend
//     compares termsVersion with its current constant.
//   - lang: the UI + mail language ("" = follow the device / English).
//     Picked by the i18n pass; the field exists now so imports and the
//     squash carry it.
//
// Both are user-writable through the normal self-update rule; nothing
// privileged lives here.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		if users.Fields.GetByName("termsAcceptedAt") == nil {
			users.Fields.Add(&core.DateField{Name: "termsAcceptedAt"})
		}
		if users.Fields.GetByName("termsVersion") == nil {
			users.Fields.Add(&core.TextField{Name: "termsVersion", Max: 16})
		}
		if users.Fields.GetByName("lang") == nil {
			users.Fields.Add(&core.SelectField{
				Name:      "lang",
				MaxSelect: 1,
				Values:    []string{"en", "de"},
			})
		}
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.Fields.RemoveByName("termsAcceptedAt")
		users.Fields.RemoveByName("termsVersion")
		users.Fields.RemoveByName("lang")
		return app.Save(users)
	})
}
