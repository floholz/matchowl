package notify

import (
	"context"
	"fmt"
	htmltemplate "html/template"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/markdown"
)

// Broadcast fans one announcement out as a real notification (email + push) to
// every eligible user, reusing the per-channel dispatch (prefs, dedup ledger,
// delivery). The dedupKey is keyed off the announcement id and the user, so
// the unique (dedupKey, channel) index makes a second "Send" a no-op rather
// than a re-spam. (Until 2026-09-28 the key had no user in it, so the index
// let exactly one recipient through per announcement.)
//
// It builds a fresh Runner (selecting the active mail/push providers) so it can
// be driven straight from the owner/admin endpoint without sharing the cron's.
func Broadcast(ctx context.Context, app core.App, ann *core.Record) (*Result, error) {
	r := New(app)
	res := &Result{}

	ncol, err := r.notificationsCol()
	if err != nil {
		return res, fmt.Errorf("notifications collection: %w", err)
	}
	recipients, err := r.eligibleUsers(readConfig(app).Allowlist)
	if err != nil {
		return res, fmt.Errorf("load recipients: %w", err)
	}

	base := r.base()
	dedupBase := "announcement:" + ann.Id + ":"
	highPriority := ann.GetBool("highPriority")
	ctaText, ctaURL := ann.GetString("ctaText"), ann.GetString("ctaUrl")
	if ctaURL == "" {
		ctaText, ctaURL = "Open Matchowl", base.url+"/"
	} else if ctaText == "" {
		ctaText = "Open"
	}
	for _, u := range recipients {
		data := tplData{
			Title:        ann.GetString("title"),
			Body:         markdown.Plain(ann.GetString("body")),
			BodyHTML:     htmltemplate.HTML(markdown.Render(ann.GetString("body"))),
			HighPriority: highPriority,
			CTAText:      ctaText,
			CTAUrl:       ctaURL,
		}
		r.dispatch(ctx, res, ncol, u, "announcement", dedupBase+u.Id, data)
	}
	return res, nil
}
