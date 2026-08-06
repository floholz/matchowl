// Command matchowl is a single-binary football prediction app: it runs the
// PocketBase backend (auth + SQLite + REST) and serves the embedded SvelteKit
// SPA from the same process, so the whole thing ships as one Docker image.
package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"

	"github.com/floholz/matchowl/internal/announce"
	"github.com/floholz/matchowl/internal/appconfig"
	"github.com/floholz/matchowl/internal/chat"
	"github.com/floholz/matchowl/internal/dev"
	"github.com/floholz/matchowl/internal/forecast"
	"github.com/floholz/matchowl/internal/leagues"
	"github.com/floholz/matchowl/internal/notify"
	"github.com/floholz/matchowl/internal/oauth"
	"github.com/floholz/matchowl/internal/players"
	"github.com/floholz/matchowl/internal/push"
	"github.com/floholz/matchowl/internal/scoring"
	"github.com/floholz/matchowl/internal/seed"
	"github.com/floholz/matchowl/internal/stats"
	"github.com/floholz/matchowl/internal/survey"
	wmsync "github.com/floholz/matchowl/internal/sync"
	"github.com/floholz/matchowl/internal/tips"
	"github.com/floholz/matchowl/internal/tournaments"
	"github.com/floholz/matchowl/internal/users"
	"github.com/floholz/matchowl/internal/web"
	_ "github.com/floholz/matchowl/migrations"
)

func main() {
	// Local dev only (MATCHOWL_DEV=1): pull ./.env into the environment so go run /
	// make run match what docker-compose injects. Must happen before anything
	// reads env (cron config, mail provider selection, etc.).
	dev.LoadDotenv()

	app := pocketbase.New()

	// Go-code migrations (collections/schema live in ./migrations).
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangGo,
		Automigrate:  true,
	})

	// Seed teams/groups/fixtures from the embedded openfootball dataset on
	// first boot (idempotent).
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := seed.Run(e.App); err != nil {
			return err
		}
		oauth.Register(e.App)
		users.Register(e.App)
		tournaments.Register(e.App, e)
		players.Register(e.App, e)
		seed.Register(e.App, e)
		wmsync.Register(e.App, e)
		leagues.Register(e.App, e)
		tips.Register(e.App, e)
		forecast.Register(e.App, e)
		scoring.Register(e.App, e)
		push.Register(e.App, e)
		notify.Register(e.App, e)
		stats.Register(e.App, e)
		announce.Register(e.App, e)
		survey.Register(e.App, e)
		appconfig.Register(e.App, e)
		chat.Register(e.App, e)
		dev.Register(e.App, e)

		// Serve the web manifest with the correct MIME so it installs as a
		// proper PWA (apis.Static would send text/plain for .webmanifest). In
		// dev (MATCHOWL_DEV=1) the manifest gets a distinct id/name/theme so the
		// browser treats the dev build as a separate installable app from prod.
		e.Router.GET("/manifest.webmanifest", func(re *core.RequestEvent) error {
			b, err := fs.ReadFile(web.DistFS(), "manifest.webmanifest")
			if err != nil {
				return apis.NewNotFoundError("", nil)
			}
			if dev.Enabled() {
				if patched, perr := devManifest(b); perr == nil {
					b = patched
				}
			}
			return re.Blob(200, "application/manifest+json", b)
		})
		return e.Next()
	})

	// Serve the embedded SvelteKit build with SPA (index.html) fallback so
	// client-side routes resolve. Registered last and only if no API/user
	// route already owns the path.
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				e.Router.GET("/{path...}", apis.Static(web.DistFS(), true))
			}
			return e.Next()
		},
		Priority: 999,
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// devManifest rewrites the PWA manifest for the dev build so it installs as a
// separate app from prod: a distinct id (the browser's identity key), a "(Dev)"
// name, and a different theme colour so it's visually obvious. Unknown fields
// are preserved.
func devManifest(b []byte) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	m["id"] = "/?dev"
	m["name"] = "Matchowl (Dev)"
	m["short_name"] = "Matchowl Dev"
	m["theme_color"] = "#ff7700"
	m["background_color"] = "#040404"
	m["icons"] = []any{
		map[string]any{"src": "/icons/maskable_icon_dev_x512.png", "sizes": "512x512", "type": "image/png", "purpose": "any maskable"},
		map[string]any{"src": "/icons/maskable_icon_dev_x192.png", "sizes": "192x192", "type": "image/png", "purpose": "any maskable"},
		map[string]any{"src": "/icons/maskable_icon_dev_x128.png", "sizes": "128x128", "type": "image/png", "purpose": "any maskable"},
		map[string]any{"src": "/icons/maskable_icon_dev_x32.png", "sizes": "32x32", "type": "image/png", "purpose": "any maskable"},
	}
	return json.Marshal(m)
}
