package sync

import (
	"testing"

	"github.com/floholz/matchowl/internal/football"
)

func TestEventKeysStableAndDistinct(t *testing.T) {
	card := football.Event{Elapsed: 30, TeamID: 1, PlayerID: 7, Type: "Card", Detail: "Yellow Card"}
	goal := football.Event{Elapsed: 41, TeamID: 2, PlayerID: 9, Type: "Goal", Detail: "Normal Goal"}
	a := eventKeys([]football.Event{card, goal, card})
	if a[0] == a[2] {
		t.Fatalf("two identical events got one key: %q", a[0])
	}
	// A cancelled goal disappears; the cards keep their keys.
	b := eventKeys([]football.Event{card, card})
	if b[0] != a[0] || b[1] != a[2] {
		t.Errorf("keys moved: %v vs %v", b, a)
	}
}
