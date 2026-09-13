package users

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
)

// One-time registration links for the closed test. While REGISTRATION_OPEN=0
// the sign-up gate (registration.go) still lets a request through when it
// carries a valid, unused link token in the X-Signup-Token header; the token
// is consumed by the account it created. Admins mint and revoke links from
// the Admin area; the register page checks a token anonymously so it can say
// "used" / "expired" before anyone types a password.

const (
	signupLinksCollection = "signup_links"
	// SignupTokenHeader carries the link token on the users create call and
	// on the OAuth2 auth call (the JS SDK forwards custom headers on both).
	SignupTokenHeader = "X-Signup-Token"
)

// Why a token is not usable; the frontend keys its message on the code.
var (
	errTokenInvalid = errors.New("invalid")
	errTokenUsed    = errors.New("used")
	errTokenExpired = errors.New("expired")
)

// findSignupLink resolves a token to a usable link, or returns why it isn't.
func findSignupLink(app core.App, token string) (*core.Record, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errTokenInvalid
	}
	rec, err := app.FindFirstRecordByData(signupLinksCollection, "token", token)
	if err != nil {
		return nil, errTokenInvalid
	}
	if rec.GetString("usedBy") != "" || !rec.GetDateTime("usedAt").IsZero() {
		return rec, errTokenUsed
	}
	if exp := rec.GetDateTime("expiresAt"); !exp.IsZero() && exp.Time().Before(time.Now()) {
		return rec, errTokenExpired
	}
	return rec, nil
}

// claimSignupLink marks the link as used before the account is created, so
// two people opening the same link at once can't both get through. release
// undoes the claim when the create call fails afterwards.
func claimSignupLink(app core.App, rec *core.Record) (release func(), err error) {
	rec.Set("usedAt", types.NowDateTime())
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return func() {
		rec.Set("usedAt", nil)
		rec.Set("usedBy", "")
		_ = app.Save(rec)
	}, nil
}

// consumeSignupLink binds the claimed link to the account it created.
func consumeSignupLink(app core.App, rec *core.Record, userID string) {
	rec.Set("usedBy", userID)
	_ = app.Save(rec)
}

// signupTokenError is the 403 body for a closed sign-up. code is
// "registration_closed" without a token, else "signup_link_<reason>".
func signupTokenError(e *core.RequestEvent, reason error) error {
	code := "registration_closed"
	msg := "registration is closed"
	switch reason {
	case errTokenUsed:
		code, msg = "signup_link_used", "this registration link has already been used"
	case errTokenExpired:
		code, msg = "signup_link_expired", "this registration link has expired"
	case errTokenInvalid:
		code, msg = "signup_link_invalid", "this registration link is not valid"
	}
	return e.JSON(http.StatusForbidden, map[string]string{"error": msg, "code": code})
}

// RegisterSignupLinks wires the link endpoints.
//
//	GET    /api/signup-links/{token}      anonymous: is this token usable?
//	GET    /api/admin/signup-links        list (newest first)
//	POST   /api/admin/signup-links        mint {label, expiresInDays}
//	DELETE /api/admin/signup-links/{id}   revoke an unused link
func RegisterSignupLinks(app core.App, se *core.ServeEvent) {
	// The register page calls this before showing the form. Anonymous on
	// purpose (the person has no account yet); it reveals nothing beyond
	// usable / used / expired for a token one already holds.
	se.Router.GET("/api/signup-links/{token}", func(e *core.RequestEvent) error {
		rec, err := findSignupLink(app, e.Request.PathValue("token"))
		if err != nil {
			return e.JSON(http.StatusOK, map[string]any{"ok": false, "code": "signup_link_" + err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true, "label": rec.GetString("label")})
	})

	g := se.Router.Group("/api/admin/signup-links")
	g.Bind(apis.RequireAuth())
	g.BindFunc(func(e *core.RequestEvent) error {
		if e.Auth == nil || !IsAdmin(e.Auth) {
			return apis.NewForbiddenError("admin only", nil)
		}
		return e.Next()
	})

	g.GET("", func(e *core.RequestEvent) error {
		recs, err := app.FindRecordsByFilter(signupLinksCollection, "id != ''", "-created", 0, 0)
		if err != nil {
			return err
		}
		if errs := app.ExpandRecords(recs, []string{"createdBy", "usedBy"}, nil); len(errs) > 0 {
			for _, err := range errs {
				return err
			}
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, signupLinkView(r))
		}
		return e.JSON(http.StatusOK, map[string]any{"links": out})
	})

	g.POST("", func(e *core.RequestEvent) error {
		var body struct {
			Label         string `json:"label"`
			ExpiresInDays int    `json:"expiresInDays"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if body.ExpiresInDays < 0 || body.ExpiresInDays > 365 {
			return apis.NewBadRequestError("expiresInDays must be between 0 (never) and 365", nil)
		}
		col, err := app.FindCollectionByNameOrId(signupLinksCollection)
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		rec.Set("token", security.RandomStringWithAlphabet(24, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ23456789"))
		rec.Set("label", strings.TrimSpace(body.Label))
		rec.Set("createdBy", e.Auth.Id)
		if body.ExpiresInDays > 0 {
			rec.Set("expiresAt", types.NowDateTime().Add(time.Duration(body.ExpiresInDays)*24*time.Hour))
		}
		if err := app.Save(rec); err != nil {
			return err
		}
		rec.Expand()["createdBy"] = e.Auth
		return e.JSON(http.StatusOK, signupLinkView(rec))
	})

	// Revoke. A used link stays as the record of who came in through it.
	g.DELETE("/{id}", func(e *core.RequestEvent) error {
		rec, err := app.FindRecordById(signupLinksCollection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such link", nil)
		}
		if rec.GetString("usedBy") != "" {
			return apis.NewBadRequestError("this link was used already; it can't be revoked", nil)
		}
		if err := app.Delete(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})
}

func signupLinkView(r *core.Record) map[string]any {
	status := "open"
	switch {
	case r.GetString("usedBy") != "" || !r.GetDateTime("usedAt").IsZero():
		status = "used"
	case !r.GetDateTime("expiresAt").IsZero() && r.GetDateTime("expiresAt").Time().Before(time.Now()):
		status = "expired"
	}
	v := map[string]any{
		"id":        r.Id,
		"token":     r.GetString("token"),
		"label":     r.GetString("label"),
		"status":    status,
		"created":   r.GetDateTime("created").Time().UTC().Format(time.RFC3339),
		"expiresAt": dateOrEmpty(r.GetDateTime("expiresAt")),
		"usedAt":    dateOrEmpty(r.GetDateTime("usedAt")),
	}
	if u := r.ExpandedOne("createdBy"); u != nil {
		v["createdBy"] = map[string]any{"id": u.Id, "name": u.GetString("name")}
	}
	if u := r.ExpandedOne("usedBy"); u != nil {
		v["usedBy"] = map[string]any{"id": u.Id, "name": u.GetString("name"), "email": u.Email()}
	}
	return v
}

func dateOrEmpty(d types.DateTime) string {
	if d.IsZero() {
		return ""
	}
	return d.Time().UTC().Format(time.RFC3339)
}
