package h2h

import (
	"fmt"
	"testing"
)

func TestJoinTakesTheGhostsPlace(t *testing.T) {
	ps := []Pair{{"a", "b"}, {"c", Ghost}}
	got := fmt.Sprint(withJoined(ps, "d"))
	if want := fmt.Sprint([]Pair{{"a", "b"}, {"c", "d"}}); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	if fmt.Sprint(ps) != fmt.Sprint([]Pair{{"a", "b"}, {"c", Ghost}}) {
		t.Fatal("input mutated")
	}
}

func TestJoinWithoutGhostPlaysTheGhost(t *testing.T) {
	got := fmt.Sprint(withJoined([]Pair{{"a", "b"}, {"c", "d"}}, "e"))
	if want := fmt.Sprint([]Pair{{"a", "b"}, {"c", "d"}, {"e", Ghost}}); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestLeaveAgainstTheGhostDropsTheDuel(t *testing.T) {
	got := fmt.Sprint(withLeft([]Pair{{"a", "b"}, {"c", Ghost}}, "c"))
	if want := fmt.Sprint([]Pair{{"a", "b"}}); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestLeaveWithoutGhostLeavesTheGhost(t *testing.T) {
	for _, tc := range []struct {
		leaver string
		want   []Pair
	}{
		{"a", []Pair{{"b", Ghost}, {"c", "d"}}},
		{"b", []Pair{{"a", Ghost}, {"c", "d"}}},
	} {
		got := fmt.Sprint(withLeft([]Pair{{"a", "b"}, {"c", "d"}}, tc.leaver))
		if want := fmt.Sprint(tc.want); got != want {
			t.Fatalf("%s leaves: got %s, want %s", tc.leaver, got, want)
		}
	}
}

func TestLeaveWithGhostSeatsTheGhostsOpponent(t *testing.T) {
	ps := []Pair{{"a", "b"}, {"c", "d"}, {"e", Ghost}}
	got := fmt.Sprint(withLeft(ps, "d"))
	if want := fmt.Sprint([]Pair{{"a", "b"}, {"c", "e"}}); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	got = fmt.Sprint(withLeft(ps, "a"))
	if want := fmt.Sprint([]Pair{{"e", "b"}, {"c", "d"}}); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestLeaveOfANonMemberChangesNothing(t *testing.T) {
	ps := []Pair{{"a", "b"}}
	if got := fmt.Sprint(withLeft(ps, "z")); got != fmt.Sprint(ps) {
		t.Fatalf("got %s", got)
	}
}

func TestCovers(t *testing.T) {
	ps := []Pair{{"a", "b"}, {"c", Ghost}}
	if !covers(ps, []string{"c", "a", "b"}) {
		t.Fatal("same members must cover")
	}
	if covers(ps, []string{"a", "b"}) || covers(ps, []string{"a", "b", "c", "d"}) || covers(ps, []string{"a", "b", "x"}) {
		t.Fatal("a different roster must not cover")
	}
	if covers([]Pair{{"a", "b"}, {"a", "c"}}, []string{"a", "b", "c"}) {
		t.Fatal("a member seated twice must not cover")
	}
}
