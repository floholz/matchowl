package markdown

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Hello **world**", "<p>Hello <strong>world</strong></p>"},
		{"a *b* and _c_", "<p>a <em>b</em> and <em>c</em></p>"},
		{"line one\nline two", "<p>line one<br>line two</p>"},
		{"# Title\n\npara", "<h1>Title</h1><p>para</p>"},
		{"- one\n- two", "<ul><li>one</li><li>two</li></ul>"},
		{"see [the app](https://play.matchowl.app) now", `<p>see <a href="https://play.matchowl.app">the app</a> now</p>`},
		{"go https://x.io/a?b=1 ok", `<p>go <a href="https://x.io/a?b=1">https://x.io/a?b=1</a> ok</p>`},
		{"[x](javascript:alert)", "<p>x</p>"},
		{"<b>raw</b>", "<p>&lt;b&gt;raw&lt;/b&gt;</p>"},
	}
	for _, c := range cases {
		if got := Render(c.in); got != c.want {
			t.Errorf("Render(%q)\n got %s\nwant %s", c.in, got, c.want)
		}
	}
}

func TestPlain(t *testing.T) {
	got := Plain("# Hi\n\nSee **this** [link](https://a.b) now")
	if !strings.Contains(got, "Hi") || strings.Contains(got, "**") || !strings.Contains(got, "link (https://a.b)") {
		t.Fatalf("Plain = %q", got)
	}
}
