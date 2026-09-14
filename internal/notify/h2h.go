package notify

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/h2h"
	"github.com/floholz/matchowl/internal/users"
)

// h2hRoundClosed tells every paired member of a just-closed round how their
// duel went — event "h2h_round", one per member and round.
func (r *Runner) h2hRoundClosed(ctx context.Context, row *core.Record) {
	res := h2h.ResultsOf(row)
	if res == nil {
		return
	}
	lg, err := r.app.FindRecordById("pools", row.GetString("pool"))
	if err != nil {
		return
	}
	ncol, err := r.notificationsCol()
	if err != nil {
		return
	}
	base := r.base()
	round := h2h.RoundName(row.GetString("label"), row.GetInt("num"))
	name := func(uid string) string {
		if uid == h2h.Ghost {
			return "the Ghost"
		}
		if u, err := r.app.FindRecordById("users", uid); err == nil {
			return u.GetString("name")
		}
		return "a former member"
	}
	tell := func(uid, rival string, mine, theirs, pts int) {
		u, err := r.app.FindRecordById("users", uid)
		if err != nil || users.IsBot(u) {
			return
		}
		data := tplData{
			League: lg.GetString("name"), Round: round, Rival: name(rival),
			Mine: mine, Theirs: theirs, Verdict: h2h.Verdict(pts),
			CTAText: "See the table", CTAUrl: base.url + "/pools/" + lg.Id,
		}
		res := &Result{}
		r.dispatch(ctx, res, ncol, u, "h2h_round", "h2h_round:"+row.Id+":"+uid, data)
	}
	for _, p := range res.Pairs {
		tell(p.A, p.B, p.ScoreA, p.ScoreB, p.PtsA)
		if p.B != h2h.Ghost {
			tell(p.B, p.A, p.ScoreB, p.ScoreA, p.PtsB)
		}
	}
}

// detectH2HBans fires "h2h_ban" at kick-off: when a match a rival banned
// for you has started, you learn it now (never before). Scans open rounds
// for bans on matches that kicked off in the last two days; the dedup key
// is the pick, so each ban tells once.
func (r *Runner) detectH2HBans(ctx context.Context, res *Result,
	recipients []*core.Record, base baseInfo) error {

	ncol, err := r.notificationsCol()
	if err != nil {
		return err
	}
	byID := map[string]*core.Record{}
	for _, u := range recipients {
		byID[u.Id] = u
	}
	now := clock.Now(r.app)
	rows, err := r.app.FindRecordsByFilter("h2h_rounds", "status = 'open'", "", 0, 0)
	if err != nil {
		return err
	}
	for _, row := range rows {
		picks, _ := r.app.FindRecordsByFilter("h2h_picks",
			"pool = {:p} && round = {:k} && kind = 'ban'", "", 0, 0,
			map[string]any{"p": row.GetString("pool"), "k": row.GetString("key")})
		if len(picks) == 0 {
			continue
		}
		var lg *core.Record
		for _, pick := range picks {
			m, err := r.app.FindRecordById("matches", pick.GetString("match"))
			if err != nil {
				continue
			}
			ko := m.GetDateTime("kickoff").Time()
			if ko.After(now) || now.Sub(ko) > 48*time.Hour {
				continue
			}
			rival, in := h2h.RivalIn(row, pick.GetString("user"))
			if !in || rival == h2h.Ghost {
				continue
			}
			u, ok := byID[rival]
			if !ok {
				continue
			}
			if lg == nil {
				if lg, err = r.app.FindRecordById("pools", row.GetString("pool")); err != nil {
					break
				}
			}
			banner := "Your rival"
			if b, err := r.app.FindRecordById("users", pick.GetString("user")); err == nil {
				banner = b.GetString("name")
			}
			teamName := func(id string) string {
				if t, err := r.app.FindRecordById("teams", id); err == nil {
					return t.GetString("name")
				}
				return "?"
			}
			data := tplData{
				League:  lg.GetString("name"),
				Round:   h2h.RoundName(row.GetString("label"), row.GetInt("num")),
				Rival:   banner,
				Match:   teamName(m.GetString("homeTeam")) + " – " + teamName(m.GetString("awayTeam")),
				CTAText: "See your duel", CTAUrl: base.url + "/pools/" + lg.Id,
			}
			r.dispatch(ctx, res, ncol, u, "h2h_ban", "h2h_ban:"+pick.Id, data)
		}
	}
	return nil
}
