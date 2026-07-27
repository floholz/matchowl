// Package appconfig exposes the handful of deploy-time settings the frontend
// needs at runtime (they live in env vars, and the SPA is a static build baked
// into the binary, so they can't be compiled in).
package appconfig

import (
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// Register wires GET /api/appconfig. Public on purpose: nothing here is a
// secret (the landing page is anonymous and may want the contact link too).
func Register(app core.App, se *core.ServeEvent) {
	se.Router.GET("/api/appconfig", func(e *core.RequestEvent) error {
		contact := strings.TrimSpace(os.Getenv("CONTACT_EMAIL"))
		if contact == "" {
			contact = "contact@floholz.dev"
		}
		return e.JSON(http.StatusOK, map[string]any{
			// Empty = no Ko-Fi configured; the support card stays hidden.
			"kofiUrl":      strings.TrimSpace(os.Getenv("KOFI_URL")),
			"contactEmail": contact,
		})
	})
}
