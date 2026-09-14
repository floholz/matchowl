package h2h

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/pools"
	"github.com/floholz/matchowl/internal/tournaments"
)

// Pick kinds. A save call doubles the match for its owner; a ban removes
// the match from the rival's round score. A ban on a save call cancels the
// double and the match counts normal ("the save call saves it").
const (
	KindSave = "save"
	KindBan  = "ban"
)

// UserPicks is one member's picks in one round.
type UserPicks struct {
	Saves []string `json:"saves"`
	Ban   string   `json:"ban,omitempty"`
}

// picksFor loads every member's picks in a round.
func picksFor(app core.App, poolID, roundKey string) map[string]*UserPicks {
	recs, _ := app.FindRecordsByFilter("h2h_picks", "pool = {:p} && round = {:r}", "created", 0, 0,
		map[string]any{"p": poolID, "r": roundKey})
	out := map[string]*UserPicks{}
	for _, r := range recs {
		uid := r.GetString("user")
		up := out[uid]
		if up == nil {
			up = &UserPicks{Saves: []string{}}
			out[uid] = up
		}
		switch r.GetString("kind") {
		case KindSave:
			up.Saves = append(up.Saves, r.GetString("match"))
		case KindBan:
			up.Ban = r.GetString("match")
		}
	}
	return out
}

func (u *UserPicks) saved(match string) bool {
	if u == nil {
		return false
	}
	for _, m := range u.Saves {
		if m == match {
			return true
		}
	}
	return false
}

// effective applies the owner's save call and the rival's ban to a
// match's base points.
func effective(base int, mine, rival *UserPicks, match string) int {
	saved := mine.saved(match)
	banned := rival != nil && rival.Ban == match
	switch {
	case saved && !banned:
		return base * 2
	case banned && !saved:
		return 0
	default:
		return base
	}
}

// PickError is a refused pick, with the HTTP status the route should use.
type PickError struct {
	Status int
	Msg    string
}

func (e *PickError) Error() string { return e.Msg }

func refuse(status int, format string, a ...any) error {
	return &PickError{Status: status, Msg: fmt.Sprintf(format, a...)}
}

// roundOf finds the playable round a match belongs to.
func roundOf(app core.App, lg *core.Record, m *core.Record) (*tournaments.Round, error) {
	if m.GetString("tournament") != pools.Season(lg) {
		return nil, refuse(400, "that match is not in this pool's season")
	}
	key := tournaments.RoundKey(m.GetString("stage"), m.GetString("roundLabel"))
	for _, r := range playable(app, lg, pools.Season(lg)) {
		if r.Key == key {
			return &r, nil
		}
	}
	return nil, refuse(409, "that matchday is not one this pool plays")
}

// rivalOf is who the member faces in a round: from the frozen pairings once
// the round has opened, else the pairing the current roster would get.
// Returns Ghost for the odd member out and "" when the member is not in
// the round at all.
func rivalOf(app core.App, lg *core.Record, roundKey string, uid string) (string, bool) {
	rows, _ := roundRows(app, lg.Id)
	var pairs []Pair
	found := false
	for _, row := range rows {
		if row.GetString("key") == roundKey {
			pairs, found = pairingsOf(row), true
			break
		}
	}
	if !found {
		pairs = Pairings(roster(app, lg.Id), len(rows))
	}
	for _, p := range pairs {
		if p.A == uid {
			return p.B, true
		}
		if p.B == uid {
			return p.A, true
		}
	}
	return "", false
}

// SetPick places (on) or removes (off) a member's pick on a match. Picks
// lock at the match's kick-off; a round that has closed takes none.
func SetPick(app core.App, lg *core.Record, uid, matchID, kind string, on bool) error {
	if lg.GetString("mode") != pools.ModeH2H {
		return refuse(400, "this pool is not head-to-head")
	}
	if kind != KindSave && kind != KindBan {
		return refuse(400, "kind: save or ban")
	}
	m, err := app.FindRecordById("matches", matchID)
	if err != nil {
		return refuse(404, "no such match")
	}
	r, err := roundOf(app, lg, m)
	if err != nil {
		return err
	}
	now := clock.Now(app)
	if !now.Before(m.GetDateTime("kickoff").Time()) {
		return refuse(409, "that match has kicked off")
	}
	if row, _ := app.FindFirstRecordByFilter("h2h_rounds", "pool = {:p} && key = {:k}",
		map[string]any{"p": lg.Id, "k": r.Key}); row != nil && row.GetString("status") == "closed" {
		return refuse(409, "that matchday is closed")
	}
	existing, _ := app.FindFirstRecordByFilter("h2h_picks",
		"pool = {:p} && user = {:u} && match = {:m} && kind = {:k}",
		map[string]any{"p": lg.Id, "u": uid, "m": matchID, "k": kind})
	if !on {
		if existing != nil {
			return app.Delete(existing)
		}
		return nil
	}
	if existing != nil {
		return nil
	}
	mine := picksFor(app, lg.Id, r.Key)[uid]
	switch kind {
	case KindSave:
		allow := lg.GetInt("saveCalls")
		if allow < 1 {
			allow = 1
		}
		if mine != nil && len(mine.Saves) >= allow {
			return refuse(409, "all %d save %s used this matchday — take one back first", allow, plural(allow, "call", "calls"))
		}
	case KindBan:
		rival, in := rivalOf(app, lg, r.Key, uid)
		if !in {
			return refuse(409, "you are not paired this matchday")
		}
		if rival == Ghost {
			return refuse(409, "no rival this matchday — the Ghost cannot be banned")
		}
		// One ban per round: move it, unless the banned match has kicked off.
		if mine != nil && mine.Ban != "" {
			old, _ := app.FindFirstRecordByFilter("h2h_picks",
				"pool = {:p} && user = {:u} && round = {:r} && kind = 'ban'",
				map[string]any{"p": lg.Id, "u": uid, "r": r.Key})
			if old != nil {
				if bm, err := app.FindRecordById("matches", old.GetString("match")); err == nil && !now.Before(bm.GetDateTime("kickoff").Time()) {
					return refuse(409, "your ban this matchday has kicked off and cannot be moved")
				}
				if err := app.Delete(old); err != nil {
					return err
				}
			}
		}
	}
	col, err := app.FindCollectionByNameOrId("h2h_picks")
	if err != nil {
		return err
	}
	rec := core.NewRecord(col)
	rec.Set("pool", lg.Id)
	rec.Set("user", uid)
	rec.Set("match", matchID)
	rec.Set("round", r.Key)
	rec.Set("kind", kind)
	return app.Save(rec)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// revealed filters a round's picks down to what everyone may see: picks
// on matches that have kicked off. A closed round is fully revealed.
func revealed(app core.App, all map[string]*UserPicks, closed bool) map[string]*UserPicks {
	if closed {
		return all
	}
	now := clock.Now(app)
	started := map[string]bool{}
	kicked := func(id string) bool {
		if v, ok := started[id]; ok {
			return v
		}
		m, err := app.FindRecordById("matches", id)
		v := err == nil && !now.Before(m.GetDateTime("kickoff").Time())
		started[id] = v
		return v
	}
	out := map[string]*UserPicks{}
	for uid, up := range all {
		r := &UserPicks{}
		for _, m := range up.Saves {
			if kicked(m) {
				r.Saves = append(r.Saves, m)
			}
		}
		if up.Ban != "" && kicked(up.Ban) {
			r.Ban = up.Ban
		}
		if len(r.Saves) > 0 || r.Ban != "" {
			out[uid] = r
		}
	}
	return out
}

// matchView is the slice of a match the picks panel renders.
func matchView(app core.App, m *core.Record, now time.Time) map[string]any {
	team := func(id string) map[string]any {
		if id == "" {
			return nil
		}
		t, err := app.FindRecordById("teams", id)
		if err != nil {
			return map[string]any{"id": id, "name": "?"}
		}
		return map[string]any{"id": id, "name": t.GetString("name"), "fifaCode": t.GetString("fifaCode"), "logo": t.GetString("logo")}
	}
	v := map[string]any{
		"id":      m.Id,
		"kickoff": iso(m.GetDateTime("kickoff").Time()),
		"status":  m.GetString("status"),
		"home":    team(m.GetString("homeTeam")),
		"away":    team(m.GetString("awayTeam")),
		"locked":  !now.Before(m.GetDateTime("kickoff").Time()),
	}
	if m.GetString("status") == "finished" || m.GetString("status") == "live" {
		v["ftHome"], v["ftAway"] = m.GetInt("ftHome"), m.GetInt("ftAway")
	}
	return v
}
