// Package markdown renders the small Markdown subset admin-written copy
// uses (announcements, mailings) into safe HTML: paragraphs, headings,
// bullet lists, bold, italic, links and bare URLs. Everything is
// HTML-escaped first, and links only keep http(s) and mailto targets, so
// nothing an admin types can inject markup. Deliberately tiny — the mail
// templates and the in-app banner need the same handful of constructs,
// and the admin app mirrors these rules for its live preview.
package markdown

import (
	"html"
	"regexp"
	"strings"
)

var (
	linkRe   = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	bareURL  = regexp.MustCompile(`(^|[\s(])(https?://[^\s<)]+)`)
	boldRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe = regexp.MustCompile(`(^|[^*\w])[*_]([^*_\n]+?)[*_]`)
)

// Render turns Markdown into HTML. Blocks are separated by blank lines;
// a single newline inside a paragraph becomes a line break.
func Render(src string) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	var out strings.Builder
	for _, block := range strings.Split(strings.TrimSpace(src), "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		switch {
		case strings.HasPrefix(block, "## "):
			out.WriteString("<h2>" + inline(strings.TrimPrefix(lines[0], "## ")) + "</h2>")
			if len(lines) > 1 {
				out.WriteString("<p>" + inlineLines(lines[1:]) + "</p>")
			}
		case strings.HasPrefix(block, "# "):
			out.WriteString("<h1>" + inline(strings.TrimPrefix(lines[0], "# ")) + "</h1>")
			if len(lines) > 1 {
				out.WriteString("<p>" + inlineLines(lines[1:]) + "</p>")
			}
		case isList(lines):
			out.WriteString("<ul>")
			for _, l := range lines {
				out.WriteString("<li>" + inline(strings.TrimSpace(l[strings.IndexAny(l, "-*")+1:])) + "</li>")
			}
			out.WriteString("</ul>")
		default:
			out.WriteString("<p>" + inlineLines(lines) + "</p>")
		}
	}
	return out.String()
}

// Plain strips the Markdown down to text (push bodies, previews, the text
// part of a mail).
func Plain(src string) string {
	s := linkRe.ReplaceAllString(src, "$1 ($2)")
	s = boldRe.ReplaceAllString(s, "$1")
	s = italicRe.ReplaceAllString(s, "$1$2")
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimLeft(l, "#")
		lines = append(lines, strings.TrimSpace(l))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func isList(lines []string) bool {
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "- ") && !strings.HasPrefix(t, "* ") {
			return false
		}
	}
	return len(lines) > 0
}

func inlineLines(lines []string) string {
	parts := make([]string, 0, len(lines))
	for _, l := range lines {
		parts = append(parts, inline(strings.TrimSpace(l)))
	}
	return strings.Join(parts, "<br>")
}

// inline escapes the text, then applies links, bare URLs, bold and italic.
func inline(s string) string {
	s = html.EscapeString(s)
	s = linkRe.ReplaceAllStringFunc(s, func(m string) string {
		g := linkRe.FindStringSubmatch(m)
		if !safeURL(g[2]) {
			return g[1]
		}
		return `<a href="` + g[2] + `">` + g[1] + `</a>`
	})
	s = bareURL.ReplaceAllStringFunc(s, func(m string) string {
		g := bareURL.FindStringSubmatch(m)
		if strings.Contains(m, "href=") {
			return m
		}
		return g[1] + `<a href="` + g[2] + `">` + g[2] + `</a>`
	})
	s = boldRe.ReplaceAllString(s, "<strong>$1</strong>")
	s = italicRe.ReplaceAllString(s, "$1<em>$2</em>")
	return s
}

func safeURL(u string) bool {
	l := strings.ToLower(u)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "mailto:")
}
