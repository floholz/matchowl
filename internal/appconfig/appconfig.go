// Package appconfig exposes the handful of deploy-time settings the frontend
// needs at runtime (they live in env vars, and the SPA is a static build baked
// into the binary, so they can't be compiled in).
package appconfig

import (
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/users"
	"github.com/floholz/matchowl/internal/version"
)

// Register wires GET /api/appconfig. Public on purpose: nothing here is a
// secret (the landing page is anonymous and may want the contact link too).
func Register(app core.App, se *core.ServeEvent) {
	se.Router.GET("/api/appconfig", func(e *core.RequestEvent) error {
		contact := strings.TrimSpace(os.Getenv("CONTACT_EMAIL"))
		if contact == "" {
			contact = "contact@floholz.dev"
		}
		location := strings.TrimSpace(os.Getenv("OPERATOR_LOCATION"))
		if location == "" {
			location = "Vienna, Austria"
		}
		return e.JSON(http.StatusOK, map[string]any{
			// Empty = no Ko-Fi configured; the support card stays hidden.
			"kofiUrl":      strings.TrimSpace(os.Getenv("KOFI_URL")),
			"contactEmail": contact,
			// Build version (release tag), shown in the Home footer and Help.
			"version": version.Short(),
			// Whether new accounts may be created (REGISTRATION_OPEN); the
			// register page and the sign-up links follow it, the server
			// enforces it (internal/users).
			"registrationOpen": users.RegistrationOpen(),
			// Who runs Matchowl, for the About & contact disclosure (Austrian
			// media law asks a private, non-commercial site for name and
			// place of residence — town, never a street address). The name
			// lives in env so it stays out of the public repository.
			"operatorName":     strings.TrimSpace(os.Getenv("OPERATOR_NAME")),
			"operatorLocation": location,
		})
	})
}
