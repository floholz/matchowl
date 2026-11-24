// Package challenges is the 1-on-1 challenge (PLAN.md, "pools at the
// center", phase 3): one friend challenges another on one match, with a
// free-text stake the app only shows. The opponent accepts before kick-off,
// both tip as usual (a tip counts wherever it applies), and the default
// scoring decides: more points wins, equal points is a draw. Picks stay
// hidden until kick-off.
//
// A pending challenge that reaches kick-off unanswered simply lapses (it's
// shown as expired; nothing is written). When the match is final the points
// are stored on the row (settledAt), which is also what the result
// notification hangs off (package notify).
package challenges

import (
	"log"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/scoring"
	"github.com/floholz/matchowl/internal/tournaments"
)

// Collection holds the challenges.
const Collection = "challenges"

// Statuses stored on a row; "expired" is derived (pending past kick-off).
const (
	Pending   = "pending"
	Accepted  = "accepted"
	Declined  = "declined"
	Cancelled = "cancelled"
	Expired   = "expired"
)

// MaxStake bounds the stake note.
const MaxStake = 140

// Outcome compares two point totals from the first side's view.
func Outcome(mine, theirs int) string {
	switch {
	case mine > theirs:
		return "won"
	case mine < theirs:
		return "lost"
	}
	return "draw"
}

// Status is a row's status with lapsed invitations shown as expired.
func Status(c, match *core.Record, now time.Time) string {
	st := c.GetString("status")
	if st == Pending && started(match, now) {
		return Expired
	}
	return st
}

func started(m *core.Record, now time.Time) bool {
	return m.GetString("status") != "scheduled" || !now.Before(m.GetDateTime("kickoff").Time())
}

// Active reports whether a challenge still binds the match to both sides
// (an open invitation or an accepted challenge).
func Active(st string) bool { return st == Pending || st == Accepted }

// MatchIDs returns the ids of the matches a user has an open or accepted
// challenge on (the "challenge" reason of package scope).
func MatchIDs(app core.App, userID string) map[string]bool {
	out := map[string]bool{}
	rows, err := app.FindRecordsByFilter(Collection,
		"(challenger = {:u} || opponent = {:u}) && (status = 'pending' || status = 'accepted')",
		"", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return out
	}
	for _, r := range rows {
		out[r.GetString("match")] = true
	}
	return out
}

// Register settles challenges when a match is final (and again on a
// correction, keeping the first settledAt).
func Register(app core.App) {
	app.OnRecordAfterUpdateSuccess("matches").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("finalizedAt") != "" {
			if err := Settle(e.App, e.Record); err != nil {
				log.Printf("[challenges] settle %s: %v", e.Record.Id, err)
			}
		}
		return e.Next()
	})
}

// Settle writes both sides' default-scoring points onto every accepted
// challenge of a finished match.
func Settle(app core.App, match *core.Record) error {
	rows, err := app.FindRecordsByFilter(Collection,
		"match = {:m} && status = 'accepted'", "", 0, 0, map[string]any{"m": match.Id})
	if err != nil || len(rows) == 0 {
		return err
	}
	cfg, err := scoring.DefaultConfig(app)
	if err != nil {
		return err
	}
	knockout := tournaments.NewStructureCache(app).IsKnockoutMatch(match)
	pts := func(uid string) int {
		tip, err := app.FindFirstRecordByFilter("tips", "user = {:u} && match = {:m}",
			map[string]any{"u": uid, "m": match.Id})
		if err != nil {
			return 0 // no tip, no points
		}
		return scoring.ScoreTip(cfg, knockout, match, tip)
	}
	for _, c := range rows {
		cp, op := pts(c.GetString("challenger")), pts(c.GetString("opponent"))
		if c.GetString("settledAt") != "" && c.GetInt("challengerPts") == cp && c.GetInt("opponentPts") == op {
			continue
		}
		c.Set("challengerPts", cp)
		c.Set("opponentPts", op)
		if c.GetString("settledAt") == "" {
			c.Set("settledAt", clock.Now(app).UTC())
		}
		if err := app.Save(c); err != nil {
			return err
		}
	}
	return nil
}

// Record is one friend's running score against me.
type Record struct {
	Won   int `json:"won"`
	Drawn int `json:"drawn"`
	Lost  int `json:"lost"`
}

// Records counts my settled challenges per friend.
func Records(app core.App, me string) map[string]*Record {
	out := map[string]*Record{}
	rows, err := app.FindRecordsByFilter(Collection,
		"(challenger = {:u} || opponent = {:u}) && status = 'accepted' && settledAt != ''",
		"", 0, 0, map[string]any{"u": me})
	if err != nil {
		return out
	}
	for _, c := range rows {
		other, mine, theirs := sides(c, me)
		r := out[other]
		if r == nil {
			r = &Record{}
			out[other] = r
		}
		switch Outcome(mine, theirs) {
		case "won":
			r.Won++
		case "lost":
			r.Lost++
		default:
			r.Drawn++
		}
	}
	return out
}

// sides returns the other user and both point totals from my side.
func sides(c *core.Record, me string) (other string, mine, theirs int) {
	if c.GetString("challenger") == me {
		return c.GetString("opponent"), c.GetInt("challengerPts"), c.GetInt("opponentPts")
	}
	return c.GetString("challenger"), c.GetInt("opponentPts"), c.GetInt("challengerPts")
}

// CleanStake trims the stake note to one line of at most MaxStake runes.
func CleanStake(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > MaxStake {
		s = string(r[:MaxStake])
	}
	return s
}

// sortViews orders challenges: soonest kick-off first for open ones.
func sortViews(v []map[string]any, newestFirst bool) {
	sort.SliceStable(v, func(i, j int) bool {
		a := v[i]["match"].(map[string]any)["kickoff"].(string)
		b := v[j]["match"].(map[string]any)["kickoff"].(string)
		if newestFirst {
			return a > b
		}
		return a < b
	})
}
