package notify

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/challenges"
)

// registerChallenges sends the three challenge events as the rows change
// (package challenges writes them): "challenge_invite" to the opponent on
// create, "challenge_accepted" to the challenger, and "challenge_result" to
// both once the match is final and the points are on the row.
func registerChallenges(app core.App, r *Runner) {
	app.OnRecordAfterCreateSuccess(challenges.Collection).BindFunc(func(e *core.RecordEvent) error {
		if !disabled() {
			c := e.Record
			go r.challengeEvent(c, "challenge_invite")
		}
		return e.Next()
	})
	app.OnRecordAfterUpdateSuccess(challenges.Collection).BindFunc(func(e *core.RecordEvent) error {
		if disabled() {
			return e.Next()
		}
		c, orig := e.Record, e.Record.Original()
		if orig == nil {
			return e.Next()
		}
		if c.GetString("status") == challenges.Accepted && orig.GetString("status") == challenges.Pending {
			go r.challengeEvent(c, "challenge_accepted")
		}
		if c.GetString("settledAt") != "" && orig.GetString("settledAt") == "" {
			go r.challengeEvent(c, "challenge_result")
		}
		return e.Next()
	})
}

func (r *Runner) challengeEvent(c *core.Record, event string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	m, err := r.app.FindRecordById("matches", c.GetString("match"))
	if err != nil {
		return
	}
	ncol, err := r.notificationsCol()
	if err != nil {
		return
	}
	teamName := func(id, label string) string {
		if t, err := r.app.FindRecordById("teams", id); err == nil {
			return t.GetString("name")
		}
		return label
	}
	base := r.base()
	match := teamName(m.GetString("homeTeam"), m.GetString("homeLabel")) + " – " + teamName(m.GetString("awayTeam"), m.GetString("awayLabel"))
	name := func(uid string) string {
		if u, err := r.app.FindRecordById("users", uid); err == nil {
			return u.GetString("name")
		}
		return "a former member"
	}
	tell := func(uid, rival string, mine, theirs int) {
		u, err := r.app.FindRecordById("users", uid)
		if err != nil {
			return
		}
		data := tplData{
			From: name(rival), Rival: name(rival), Match: match, Stake: c.GetString("stake"),
			WhenText: formatKickoff(m.GetDateTime("kickoff").Time()),
			Mine:     mine, Theirs: theirs,
			CTAText: "Open the match", CTAUrl: base.url + "/m/" + m.Id,
		}
		switch challenges.Outcome(mine, theirs) {
		case "won":
			data.Verdict = "beat"
		case "lost":
			data.Verdict = "lost to"
		default:
			data.Verdict = "drew with"
		}
		res := &Result{}
		r.dispatch(ctx, res, ncol, u, event, event+":"+c.Id+":"+uid, data)
	}
	ch, op := c.GetString("challenger"), c.GetString("opponent")
	switch event {
	case "challenge_invite":
		tell(op, ch, 0, 0)
	case "challenge_accepted":
		tell(ch, op, 0, 0)
	case "challenge_result":
		tell(ch, op, c.GetInt("challengerPts"), c.GetInt("opponentPts"))
		tell(op, ch, c.GetInt("opponentPts"), c.GetInt("challengerPts"))
	}
}
