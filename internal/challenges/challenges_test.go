package challenges

import (
	"strings"
	"testing"
)

func TestOutcome(t *testing.T) {
	for _, c := range []struct {
		mine, theirs int
		want         string
	}{{3, 1, "won"}, {0, 2, "lost"}, {1, 1, "draw"}, {0, 0, "draw"}} {
		if got := Outcome(c.mine, c.theirs); got != c.want {
			t.Errorf("Outcome(%d, %d) = %s, want %s", c.mine, c.theirs, got, c.want)
		}
	}
}

func TestCleanStake(t *testing.T) {
	if got := CleanStake("  loser buys\n the next   round "); got != "loser buys the next round" {
		t.Errorf("got %q", got)
	}
	if got := CleanStake(strings.Repeat("ä", 200)); len([]rune(got)) != MaxStake {
		t.Errorf("len = %d, want %d", len([]rune(got)), MaxStake)
	}
}
