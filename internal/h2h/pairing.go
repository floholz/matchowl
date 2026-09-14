// Package h2h runs head-to-head pools: every matchday (round) of the
// pool's season pairs the members up by a fixed rotation, each duel is
// decided by the two members' tip points over that round's matches, and
// the pool's table counts wins, draws and losses (3 / 1 / 0). See PLAN.md,
// "head-to-head pools".
package h2h

// Ghost is the opponent of the odd member out: it scores the rounded mean
// of the other members' round scores, so everyone plays every round.
const Ghost = ""

// Pair is one duel of a round. B is Ghost when the roster is odd.
type Pair struct {
	A string `json:"a"`
	B string `json:"b"`
}

// Pairings pairs a roster for the round with the given ordinal (0-based
// count of rounds the pool has played) by the circle method: one member
// stays fixed, the others rotate one step per round, so the schedule is
// predictable and repeats in the same order once every member has met
// every other. An odd roster gets the Ghost as its last member. The
// roster must be in a stable order (the caller sorts by join time).
func Pairings(roster []string, ordinal int) []Pair {
	n := len(roster)
	if n == 0 {
		return nil
	}
	seq := make([]string, 0, n+1)
	seq = append(seq, roster...)
	if n%2 == 1 {
		seq = append(seq, Ghost)
		n++
	}
	if ordinal < 0 {
		ordinal = 0
	}
	// Rotate everyone but the first by ordinal steps.
	rot := make([]string, n-1)
	k := ordinal % (n - 1)
	for i := 0; i < n-1; i++ {
		rot[(i+k)%(n-1)] = seq[i+1]
	}
	seq = append(seq[:1], rot...)
	out := make([]Pair, 0, n/2)
	for i := 0; i < n/2; i++ {
		a, b := seq[i], seq[n-1-i]
		if a == Ghost { // keep the ghost on the B side
			a, b = b, a
		}
		out = append(out, Pair{A: a, B: b})
	}
	return out
}

// Outcome points: win 3, draw 1, loss 0.
func outcome(mine, theirs int) int {
	switch {
	case mine > theirs:
		return 3
	case mine == theirs:
		return 1
	default:
		return 0
	}
}
