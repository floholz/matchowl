package users

import (
	"log"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// EnsureSuperuser creates the PocketBase superuser named by PB_ADMIN_EMAIL
// and PB_ADMIN_PASSWORD when both are set and no superuser has that email
// yet. An existing one is left alone (its password may have been changed
// since). Runs at boot and from the dev seed, so a fresh database comes
// with its admin.
func EnsureSuperuser(app core.App) {
	email := strings.TrimSpace(os.Getenv("PB_ADMIN_EMAIL"))
	pass := os.Getenv("PB_ADMIN_PASSWORD")
	if email == "" || pass == "" {
		return
	}
	if _, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email); err == nil {
		return
	}
	col, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		log.Printf("[users] superuser: %v", err)
		return
	}
	rec := core.NewRecord(col)
	rec.SetEmail(email)
	rec.SetPassword(pass)
	if err := app.Save(rec); err != nil {
		log.Printf("[users] superuser %s: %v", email, err)
		return
	}
	log.Printf("[users] created superuser %s (PB_ADMIN_EMAIL)", email)
}
