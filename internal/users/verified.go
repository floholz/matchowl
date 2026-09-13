package users

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Verification tiers (decided 2026-09-13): anyone may register with any
// email, and an unverified account can tip, play competitions and forecast
// — everything that is private to the account. It receives no mail at all
// (see internal/notify) and no social features: friends, pools, chat, people
// search and the everyone board wait until the address is verified. Google
// sign-in counts as verified (PocketBase marks it).

// ErrUnverified is the JSON body the social endpoints answer with; the
// frontend keys its "verify first" states on the code.
const unverifiedCode = "unverified"

// RequireVerified is a route-group middleware that rejects unverified
// accounts with 403 + {"error", "code": "unverified"}. It runs after
// apis.RequireAuth, so e.Auth is set. Paths listed in exempt (exact match)
// pass through — for reads that should stay quiet, like "my pools".
func RequireVerified(exempt ...string) *hook.Handler[*core.RequestEvent] {
	skip := make(map[string]bool, len(exempt))
	for _, p := range exempt {
		skip[p] = true
	}
	return &hook.Handler[*core.RequestEvent]{
		Id: "matchowlRequireVerified",
		Func: func(e *core.RequestEvent) error {
			if e.Auth == nil || e.Auth.Verified() || e.HasSuperuserAuth() || skip[e.Request.URL.Path] {
				return e.Next()
			}
			return e.JSON(http.StatusForbidden, map[string]string{
				"error": "verify your email address to use friends and pools",
				"code":  unverifiedCode,
			})
		},
	}
}

// Verified reports whether a request's account may use social features.
func Verified(e *core.RequestEvent) bool {
	return e.Auth != nil && (e.Auth.Verified() || e.HasSuperuserAuth())
}

// purgeDays is how long an unverified account lives before the nightly
// sweep deletes it. Long on purpose: people come back; a manual sweep can
// always shorten it if throwaway accounts pile up.
const purgeDays = 180

// RegisterPurge schedules the nightly deletion of unverified, non-bot
// accounts older than purgeDays (UNVERIFIED_PURGE_DAYS overrides; 0 turns
// the sweep off). Deleting the user cascades to tips, forecasts, memberships
// and the rest via the relation fields.
func RegisterPurge(app core.App) {
	days := purgeDays
	if v := os.Getenv("UNVERIFIED_PURGE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			days = n
		}
	}
	if days <= 0 {
		log.Printf("[users] unverified-account purge disabled")
		return
	}
	app.Cron().MustAdd("purge-unverified", "20 4 * * *", func() {
		n, err := PurgeUnverified(app, days)
		if err != nil {
			log.Printf("[users] purge failed: %v", err)
			return
		}
		if n > 0 {
			log.Printf("[users] purged %d unverified accounts older than %d days", n, days)
		}
	})
	log.Printf("[users] unverified-account purge enabled (after %d days, nightly)", days)
}

// PurgeUnverified deletes unverified, non-bot users created more than days
// ago and returns how many went.
func PurgeUnverified(app core.App, days int) (int, error) {
	cutoff := types.NowDateTime().Add(-time.Duration(days) * 24 * time.Hour)
	recs, err := app.FindRecordsByFilter("users",
		"verified = false && role != 'bot' && created < {:cutoff}", "", 0, 0,
		map[string]any{"cutoff": cutoff})
	if err != nil {
		return 0, err
	}
	n := 0
	for _, u := range recs {
		if err := app.Delete(u); err != nil {
			log.Printf("[users] purge: delete %s failed: %v", u.Id, err)
			continue
		}
		n++
	}
	return n, nil
}
