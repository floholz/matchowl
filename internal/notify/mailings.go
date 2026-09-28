package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/markdown"
	"github.com/floholz/matchowl/internal/users"
)

// Mailings (2026-09-28): one targeted email to an audience, written in the
// admin app — subject, Markdown body, a call-to-action, the audience
// filter — sent through the same dispatch as every notification (prefs,
// dedup ledger, provider), with the mailing itself keeping what happened.

const mailingsCollection = "mailings"

// EventMailing is the ledger event (and the notification preference key)
// for mailings — users can opt out of them like any other mail.
const EventMailing = "mailing"

func mailingView(r *core.Record) map[string]any {
	var aud Audience
	_ = r.UnmarshalJSONField("audience", &aud)
	var res any
	_ = r.UnmarshalJSONField("result", &res)
	sentAt := ""
	if t := r.GetDateTime("sentAt").Time(); !t.IsZero() {
		sentAt = t.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"id": r.Id, "subject": r.GetString("subject"), "body": r.GetString("body"),
		"ctaText": r.GetString("ctaText"), "ctaUrl": r.GetString("ctaUrl"),
		"audience": aud, "status": r.GetString("status"), "sentAt": sentAt,
		"recipients": r.GetInt("recipients"), "result": res,
		"created": r.GetDateTime("created").Time().UTC().Format(time.RFC3339),
		"updated": r.GetDateTime("updated").Time().UTC().Format(time.RFC3339),
	}
}

// mailingData is the template data for one mailing (the recipient's
// first name is available as {{.Name}} in the body through the renderer).
func (r *Runner) mailingData(subject, body, ctaText, ctaUrl string, u *core.Record) tplData {
	name := ""
	if u != nil {
		name = strings.TrimSpace(strings.SplitN(u.GetString("name"), " ", 2)[0])
	}
	text := strings.ReplaceAll(body, "{{name}}", name)
	d := tplData{
		Title:    strings.ReplaceAll(subject, "{{name}}", name),
		Body:     markdown.Plain(text),
		BodyHTML: htmltemplate.HTML(markdown.Render(text)),
		CTAText:  ctaText,
		CTAUrl:   ctaUrl,
	}
	if d.CTAUrl == "" {
		d.CTAText, d.CTAUrl = "Open Matchowl", r.base().url+"/"
	} else if d.CTAText == "" {
		d.CTAText = "Open"
	}
	d.AppName = d.AppNameOr(r.base().appName)
	d.BaseURL = r.base().url
	d.SettingsUrl = r.base().url + "/settings"
	return d
}

// SendMailing fans one mailing out to its audience. Idempotent per
// mailing and user: the ledger keys off both.
func SendMailing(ctx context.Context, app core.App, rec *core.Record, by string) (*Result, int, error) {
	r := New(app)
	res := &Result{}
	ncol, err := r.notificationsCol()
	if err != nil {
		return res, 0, err
	}
	var aud Audience
	_ = rec.UnmarshalJSONField("audience", &aud)
	recipients, err := aud.Resolve(app)
	if err != nil {
		return res, 0, err
	}
	for _, u := range recipients {
		data := r.mailingData(rec.GetString("subject"), rec.GetString("body"), rec.GetString("ctaText"), rec.GetString("ctaUrl"), u)
		// The ledger's unique key is (dedupKey, channel): one key per user.
		r.dispatchEmail(ctx, res, ncol, u, EventMailing, "mailing:"+rec.Id+":"+u.Id, data)
	}
	rec.Set("status", "sent")
	rec.Set("sentAt", time.Now().UTC())
	rec.Set("sentBy", by)
	rec.Set("recipients", len(recipients))
	b, _ := json.Marshal(res)
	rec.Set("result", string(b))
	if err := app.Save(rec); err != nil {
		return res, len(recipients), err
	}
	return res, len(recipients), nil
}

// registerMailings wires the admin endpoints.
func registerMailings(app core.App, se *core.ServeEvent) {
	g := se.Router.Group("/api/admin/mailings")
	g.Bind(apis.RequireAuth())
	g.BindFunc(func(e *core.RequestEvent) error {
		if e.Auth == nil || !users.IsAdmin(e.Auth) {
			return apis.NewForbiddenError("admin only", nil)
		}
		return e.Next()
	})
	type body struct {
		Subject  *string   `json:"subject"`
		Body     *string   `json:"body"`
		CTAText  *string   `json:"ctaText"`
		CTAUrl   *string   `json:"ctaUrl"`
		Audience *Audience `json:"audience"`
	}
	apply := func(rec *core.Record, b body) error {
		if b.Subject != nil {
			rec.Set("subject", strings.TrimSpace(*b.Subject))
		}
		if b.Body != nil {
			rec.Set("body", strings.TrimSpace(*b.Body))
		}
		if b.CTAText != nil {
			rec.Set("ctaText", strings.TrimSpace(*b.CTAText))
		}
		if b.CTAUrl != nil {
			u := strings.TrimSpace(*b.CTAUrl)
			if u != "" && !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
				return apis.NewBadRequestError("the button link must start with http:// or https://", nil)
			}
			rec.Set("ctaUrl", u)
		}
		if b.Audience != nil {
			j, _ := json.Marshal(b.Audience)
			rec.Set("audience", string(j))
		}
		if rec.GetString("subject") == "" || rec.GetString("body") == "" {
			return apis.NewBadRequestError("subject and body are required", nil)
		}
		return nil
	}
	find := func(e *core.RequestEvent) (*core.Record, error) {
		rec, err := app.FindRecordById(mailingsCollection, e.Request.PathValue("id"))
		if err != nil {
			return nil, apis.NewNotFoundError("no such mailing", nil)
		}
		return rec, nil
	}

	// GET /api/admin/mailings — newest first.
	g.GET("", func(e *core.RequestEvent) error {
		recs, err := app.FindRecordsByFilter(mailingsCollection, "id != ''", "-created", 0, 0)
		if err != nil {
			return err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, mailingView(r))
		}
		return e.JSON(http.StatusOK, map[string]any{"mailings": out})
	})
	// POST /api/admin/mailings — create a draft.
	g.POST("", func(e *core.RequestEvent) error {
		var b body
		if err := e.BindBody(&b); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		col, err := app.FindCollectionByNameOrId(mailingsCollection)
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		rec.Set("status", "draft")
		if err := apply(rec, b); err != nil {
			return err
		}
		if err := app.Save(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, mailingView(rec))
	})
	// POST /api/admin/mailings/{id} — edit a draft (a sent mailing is history).
	g.POST("/{id}", func(e *core.RequestEvent) error {
		rec, err := find(e)
		if err != nil {
			return err
		}
		if rec.GetString("status") == "sent" {
			return apis.NewBadRequestError("this mailing was sent; duplicate it instead", nil)
		}
		var b body
		if err := e.BindBody(&b); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		if err := apply(rec, b); err != nil {
			return err
		}
		if err := app.Save(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, mailingView(rec))
	})
	g.DELETE("/{id}", func(e *core.RequestEvent) error {
		rec, err := find(e)
		if err != nil {
			return err
		}
		if err := app.Delete(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"ok": true})
	})
	// POST /api/admin/mailings/audience { audience } — who it reaches now.
	g.POST("/audience", func(e *core.RequestEvent) error {
		var b struct {
			Audience Audience `json:"audience"`
		}
		if err := e.BindBody(&b); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		recs, err := b.Audience.Resolve(app)
		if err != nil {
			return err
		}
		out := make([]Recipient, 0, len(recs))
		for _, u := range recs {
			out = append(out, Recipient{ID: u.Id, Name: u.GetString("name"), Email: u.Email()})
		}
		return e.JSON(http.StatusOK, map[string]any{"count": len(out), "recipients": out})
	})
	// POST /api/admin/mailings/render { subject, body, ctaText, ctaUrl } —
	// the mail as the recipient will see it (for the live preview).
	g.POST("/render", func(e *core.RequestEvent) error {
		var b struct {
			Subject, Body, CTAText, CTAUrl string
			Event                          string // "mailing" (default) or "announcement"
		}
		if err := e.BindBody(&b); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		event := EventMailing
		if b.Event == "announcement" {
			event = "announcement"
		}
		r := New(app)
		data := r.mailingData(b.Subject, b.Body, b.CTAText, b.CTAUrl, e.Auth)
		subject, html, text, err := render(event, data)
		if err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		// The brand mark is an inline attachment in real mail; the preview
		// gets it as a data URI so the iframe shows it too.
		html = strings.ReplaceAll(html, "cid:mark", "data:image/png;base64,"+base64.StdEncoding.EncodeToString(markPNG))
		return e.JSON(http.StatusOK, map[string]any{"subject": subject, "html": html, "text": text})
	})
	// POST /api/admin/mailings/{id}/test — the mailing to the caller only.
	g.POST("/{id}/test", func(e *core.RequestEvent) error {
		rec, err := find(e)
		if err != nil {
			return err
		}
		r := New(app)
		data := r.mailingData(rec.GetString("subject"), rec.GetString("body"), rec.GetString("ctaText"), rec.GetString("ctaUrl"), e.Auth)
		subject, html, text, err := render(EventMailing, data)
		if err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		ctx, cancel := context.WithTimeout(e.Request.Context(), 30*time.Second)
		defer cancel()
		if _, err := r.sender.Send(ctx, mailerMessage(e.Auth, "[test] "+subject, html, text)); err != nil {
			return apis.NewApiError(http.StatusBadGateway, err.Error(), nil)
		}
		return e.JSON(http.StatusOK, map[string]any{"to": e.Auth.Email(), "provider": r.sender.Name()})
	})
	// POST /api/admin/mailings/{id}/send — to the audience, once.
	g.POST("/{id}/send", func(e *core.RequestEvent) error {
		rec, err := find(e)
		if err != nil {
			return err
		}
		if rec.GetString("status") == "sent" {
			return apis.NewBadRequestError("already sent", nil)
		}
		ctx, cancel := context.WithTimeout(e.Request.Context(), 5*time.Minute)
		defer cancel()
		res, n, err := SendMailing(ctx, app, rec, e.Auth.Id)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, fmt.Sprintf("send: %v", err), nil)
		}
		return e.JSON(http.StatusOK, map[string]any{"mailing": mailingView(rec), "recipients": n, "result": res})
	})
}
