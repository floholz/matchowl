package h2h

import (
	"fmt"
	"sort"
	"testing"
)

func key(p Pair) string {
	a, b := p.A, p.B
	if a > b {
		a, b = b, a
	}
	return a + "-" + b
}

func TestPairingsEvenRosterIsARoundRobin(t *testing.T) {
	roster := []string{"a", "b", "c", "d", "e", "f"}
	seen := map[string]int{}
	for r := 0; r < 5; r++ {
		ps := Pairings(roster, r)
		if len(ps) != 3 {
			t.Fatalf("round %d: %d pairs", r, len(ps))
		}
		used := map[string]bool{}
		for _, p := range ps {
			if used[p.A] || used[p.B] || p.A == p.B {
				t.Fatalf("round %d: %v pairs someone twice", r, ps)
			}
			used[p.A], used[p.B] = true, true
			seen[key(p)]++
		}
	}
	if len(seen) != 15 {
		t.Fatalf("5 rounds of 6 must produce all 15 duels, got %d", len(seen))
	}
	for k, n := range seen {
		if n != 1 {
			t.Fatalf("duel %s played %d times", k, n)
		}
	}
	// Round 5 repeats round 0.
	if fmt.Sprint(Pairings(roster, 5)) != fmt.Sprint(Pairings(roster, 0)) {
		t.Fatal("schedule must repeat after n-1 rounds")
	}
}

func TestPairingsOddRosterUsesGhost(t *testing.T) {
	roster := []string{"a", "b", "c"}
	ghosted := []string{}
	for r := 0; r < 3; r++ {
		ps := Pairings(roster, r)
		if len(ps) != 2 {
			t.Fatalf("round %d: %v", r, ps)
		}
		for _, p := range ps {
			if p.A == Ghost {
				t.Fatalf("ghost must be on the B side: %v", ps)
			}
			if p.B == Ghost {
				ghosted = append(ghosted, p.A)
			}
		}
	}
	sort.Strings(ghosted)
	if fmt.Sprint(ghosted) != "[a b c]" {
		t.Fatalf("each member plays the ghost once in 3 rounds, got %v", ghosted)
	}
}

func TestPairingsTinyRosters(t *testing.T) {
	if ps := Pairings(nil, 0); ps != nil {
		t.Fatalf("empty roster: %v", ps)
	}
	if ps := Pairings([]string{"solo"}, 4); len(ps) != 1 || ps[0].A != "solo" || ps[0].B != Ghost {
		t.Fatalf("single member plays the ghost: %v", ps)
	}
	if ps := Pairings([]string{"a", "b"}, 7); len(ps) != 1 || ps[0].A != "a" || ps[0].B != "b" {
		t.Fatalf("two members always meet: %v", ps)
	}
}

func TestOutcome(t *testing.T) {
	if outcome(3, 1) != 3 || outcome(2, 2) != 1 || outcome(0, 5) != 0 {
		t.Fatal("3 / 1 / 0")
	}
}
