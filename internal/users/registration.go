package users

import (
	"errors"
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
// still create accounts (the dashboard, imports); and a request carrying a
// usable one-time link token (signup_links.go) gets through once.
func RegisterSignupGate(app core.App) {
	app.OnRecordCreateRequest("users").BindFunc(func(e *core.RecordRequestEvent) error {
		if RegistrationOpen() || e.HasSuperuserAuth() {
			return e.Next()
		}
		// A Google sign-up creates its record through an internal create
		// request (same headers, OAuth2 context); the OAuth2 hook below has
		// gated and claimed the link already, so don't judge it twice.
		if info, _ := e.RequestInfo(); info != nil && info.Context == core.RequestInfoContextOAuth2 {
			return e.Next()
		}
		link, release, err := gateWithLink(app, e.RequestEvent)
		if errors.Is(err, errGateRefused) {
			return nil // 403 written
		}
		if err != nil {
			return err
		}
		if err := e.Next(); err != nil {
			release()
			return err
		}
		consumeSignupLink(app, link, e.Record.Id)
		return nil
	})
	app.OnRecordAuthWithOAuth2Request("users").BindFunc(func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		if RegistrationOpen() || !e.IsNewRecord {
			return e.Next()
		}
		link, release, err := gateWithLink(app, e.RequestEvent)
		if errors.Is(err, errGateRefused) {
			return nil // 403 written
		}
		if err != nil {
			return err
		}
		if err := e.Next(); err != nil {
			release()
			return err
		}
		if e.Record != nil {
			consumeSignupLink(app, link, e.Record.Id)
		}
		return nil
	})
}

// errGateRefused: the 403 has been written; the hook must stop without
// calling Next and without returning an error of its own.
var errGateRefused = errors.New("sign-up refused")

// gateWithLink answers a closed-registration request: without a usable
// token in the header it writes the 403 and returns errGateRefused; with
// one it claims the link and hands back the release for a failed create.
func gateWithLink(app core.App, e *core.RequestEvent) (*core.Record, func(), error) {
	token := e.Request.Header.Get(SignupTokenHeader)
	if token == "" {
		return nil, nil, refuse(signupTokenError(e, nil))
	}
	link, err := findSignupLink(app, token)
	if err != nil {
		return nil, nil, refuse(signupTokenError(e, err))
	}
	release, err := claimSignupLink(app, link)
	if err != nil {
		return nil, nil, err
	}
	return link, release, nil
}

// refuse turns the outcome of writing the 403 into the sentinel the hooks
// key on (a write failure surfaces as itself).
func refuse(writeErr error) error {
	if writeErr != nil {
		return writeErr
	}
	return errGateRefused
}
