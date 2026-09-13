package mailer

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// logSender is the no-op fallback used in development or when no real provider
// is configured. It logs a one-line summary (not the full body) plus any
// action links the mail carries, so verification / reset / email-change
// flows can be followed from the log, and returns a synthetic message id so
// the rest of the pipeline behaves as if a send succeeded.
type logSender struct {
	from from
	seq  int
}

func newLog(f from) *logSender { return &logSender{from: f} }

func (l *logSender) Name() string { return "log" }

var linkRe = regexp.MustCompile(`https?://[^\s"'<>)]+`)

func (l *logSender) Send(_ context.Context, m Message) (string, error) {
	l.seq++
	log.Printf("[mailer:log] to=%q subject=%q (not actually sent)", m.ToEmail, m.Subject)
	for _, u := range links(m) {
		log.Printf("[mailer:log]   link: %s", u)
	}
	return fmt.Sprintf("log-%d", l.seq), nil
}

// links returns the distinct URLs in the mail body (text first, else HTML),
// HTML entities unescaped, static assets skipped.
func links(m Message) []string {
	body := m.Text
	if strings.TrimSpace(body) == "" {
		body = m.HTML
	}
	seen := map[string]bool{}
	var out []string
	for _, u := range linkRe.FindAllString(body, -1) {
		u = strings.ReplaceAll(u, "&amp;", "&")
		u = strings.TrimRight(u, ".,;")
		if seen[u] || strings.Contains(u, "/api/files/") || strings.HasPrefix(u, "https://fonts.") {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}
