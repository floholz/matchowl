package users

import (
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// RegistrationOpen reports whether new accounts may be created. Controlled
// by REGISTRATION_OPEN: "0" / "false" / "no" / "closed" closes sign-up (a
// private test run); anything else, including unset, keeps it open.
func RegistrationOpen() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("REGISTRATION_OPEN"))) {
	case "0", "false", "no", "off", "closed":
		return false
	}
	return true
}

// RegisterSignupGate refuses new accounts while registration is closed —
// both the email/password create call and a Google sign-in that would
// create a record. Existing accounts sign in as usual; superusers may
// still create accounts (the dashboard, imports).
func RegisterSignupGate(app core.App) {
	app.OnRecordCreateRequest("users").BindFunc(func(e *core.RecordRequestEvent) error {
		if !RegistrationOpen() && !e.HasSuperuserAuth() {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "registration is closed",
				"code":  "registration_closed",
			})
		}
		return e.Next()
	})
	app.OnRecordAuthWithOAuth2Request("users").BindFunc(func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		if !RegistrationOpen() && e.IsNewRecord {
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "registration is closed",
				"code":  "registration_closed",
			})
		}
		return e.Next()
	})
}
