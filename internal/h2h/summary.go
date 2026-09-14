package h2h

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

var trailingNumber = regexp.MustCompile(`\d+\s*$`)

// RoundName is how a round is spoken of: "Matchday 12" for numbered league
// rounds, the label itself otherwise ("Quarter-finals").
func RoundName(label string, num int) string {
	if num > 0 && trailingNumber.MatchString(label) {
		return fmt.Sprintf("Matchday %d", num)
	}
	return label
}

// PairingsOf exposes a round row's frozen pairings.
func PairingsOf(row *core.Record) []Pair { return pairingsOf(row) }

// RivalIn is who the member faces in a round row: Ghost for the odd one
// out, "" (and false) when the member is not in the round.
func RivalIn(row *core.Record, uid string) (string, bool) {
	for _, p := range pairingsOf(row) {
		if p.A == uid {
			return p.B, true
		}
		if p.B == uid {
			return p.A, true
		}
	}
	return "", false
}

// Verdict words a duel from A's side: "beat", "drew with", "lost to".
func Verdict(ptsA int) string {
	switch ptsA {
	case 3:
		return "beat"
	case 1:
		return "drew with"
	default:
		return "lost to"
	}
}

// Summary lines a closed round's duels: "Anna beat Flo 14–9 · Kai drew
// with Jordan 8–8 · Pat lost to the Ghost 5–9".
func Summary(app core.App, res *Results) string {
	names := map[string]string{}
	name := func(uid string) string {
		if uid == Ghost {
			return "the Ghost"
		}
		if n, ok := names[uid]; ok {
			return n
		}
		n := "a former member"
		if u, err := app.FindRecordById("users", uid); err == nil {
			n = u.GetString("name")
		}
		names[uid] = n
		return n
	}
	parts := make([]string, 0, len(res.Pairs))
	for _, p := range res.Pairs {
		// Word it from the winner's side so "beat" leads where there is one.
		a, b, sa, sb, pa := p.A, p.B, p.ScoreA, p.ScoreB, p.PtsA
		if p.PtsB == 3 {
			a, b, sa, sb, pa = p.B, p.A, p.ScoreB, p.ScoreA, p.PtsB
		}
		parts = append(parts, fmt.Sprintf("%s %s %s %d–%d", name(a), Verdict(pa), name(b), sa, sb))
	}
	return strings.Join(parts, " · ")
}

// postSummary drops the round's results into the pool chat as a system
// message, the banter opener.
func postSummary(app core.App, lg *core.Record, row *core.Record, res *Results) {
	col, err := app.FindCollectionByNameOrId("pool_messages")
	if err != nil {
		return
	}
	text := RoundName(row.GetString("label"), row.GetInt("num")) + " is in · " + Summary(app, res)
	if len(res.Pairs) == 0 {
		text = RoundName(row.GetString("label"), row.GetInt("num")) + " is in — nobody was paired."
	}
	msg := core.NewRecord(col)
	msg.Set("pool", lg.Id)
	msg.Set("text", text)
	msg.Set("system", true)
	if err := app.Save(msg); err != nil {
		log.Printf("[h2h] chat summary for pool %s: %v", lg.Id, err)
	}
}
