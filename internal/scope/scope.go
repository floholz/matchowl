// Package scope is the one server-side definition of "my matches" (PLAN.md,
// "pools at the center"). Home, the Matches filter and notifications all
// read it. A match is mine when it is
//
//  1. a match of a season one of my pools binds, from that pool's start
//     date on;
//  2. a match of a team I follow (phase 2);
//  3. a match I have a challenge on (phase 3);
//  4. a match I have tipped.
//
// Nobody "plays" a competition any more: any covered match can be tipped,
// and a tip counts wherever it applies.
package scope

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/tournaments"
)

// Why a match is in someone's scope. A row can carry several.
const (
	ReasonPool = "pool"
	ReasonTip  = "tip"
)

// globalInviteCode marks the auto-managed Global pool, which binds no season
// (mirrors pools.GlobalInviteCode; pools imports this package's callers).
const globalInviteCode = "GLOBAL"

// Scope is one user's "my matches", resolved once per request.
type Scope struct {
	// Seasons maps a tournament id to the earliest kick-off that counts
	// (zero: the whole season). Several pools on one season keep the
	// earliest start.
	Seasons map[string]time.Time
	// Tipped holds the ids of every match the user has tipped.
	Tipped map[string]bool
}

// Mine resolves a user's scope.
func Mine(app core.App, userID string) (*Scope, error) {
	s := &Scope{Seasons: map[string]time.Time{}, Tipped: map[string]bool{}}
	if userID == "" {
		return s, nil
	}
	mems, err := app.FindRecordsByFilter("pool_members",
		"user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}
	for _, m := range mems {
		lg, err := app.FindRecordById("pools", m.GetString("pool"))
		if err != nil || lg.GetString("inviteCode") == globalInviteCode {
			continue
		}
		since := lg.GetDateTime("startDate").Time()
		for _, tid := range lg.GetStringSlice("tournaments") {
			t, err := app.FindRecordById("tournaments", tid)
			if err != nil {
				continue
			}
			if st := t.GetString("status"); st == tournaments.StatusDraft || st == tournaments.StatusArchived {
				continue
			}
			if cur, ok := s.Seasons[tid]; !ok || since.IsZero() || (!cur.IsZero() && since.Before(cur)) {
				s.Seasons[tid] = since
			}
		}
	}
	tips, err := app.FindRecordsByFilter("tips",
		"user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}
	for _, t := range tips {
		s.Tipped[t.GetString("match")] = true
	}
	return s, nil
}

// Reasons lists why a match is in the scope; empty when it isn't.
func (s *Scope) Reasons(m *core.Record) []string {
	var out []string
	if since, ok := s.Seasons[m.GetString("tournament")]; ok {
		if since.IsZero() || !m.GetDateTime("kickoff").Time().Before(since) {
			out = append(out, ReasonPool)
		}
	}
	if s.Tipped[m.Id] {
		out = append(out, ReasonTip)
	}
	return out
}

// Has reports whether a match is in the scope.
func (s *Scope) Has(m *core.Record) bool { return len(s.Reasons(m)) > 0 }

// HasPool reports whether a match is in the scope through a pool (the
// matches Home asks you to tip; a tip on its own doesn't ask for more).
func (s *Scope) HasPool(m *core.Record) bool {
	for _, r := range s.Reasons(m) {
		if r == ReasonPool {
			return true
		}
	}
	return false
}

// Matches returns the scope's matches kicking off in [from, to), sorted by
// kick-off then number. Zero bounds are open.
func (s *Scope) Matches(app core.App, from, to time.Time) ([]*core.Record, error) {
	ids := make([]any, 0, len(s.Seasons))
	for tid := range s.Seasons {
		ids = append(ids, tid)
	}
	tipped := make([]any, 0, len(s.Tipped))
	for mid := range s.Tipped {
		tipped = append(tipped, mid)
	}
	if len(ids) == 0 && len(tipped) == 0 {
		return nil, nil
	}
	var cond []dbx.Expression
	if len(ids) > 0 {
		cond = append(cond, dbx.In("tournament", ids...))
	}
	if len(tipped) > 0 {
		cond = append(cond, dbx.In("id", tipped...))
	}
	q := app.RecordQuery("matches").AndWhere(dbx.Or(cond...))
	if !from.IsZero() {
		q = q.AndWhere(dbx.NewExp("kickoff >= {:from}", dbx.Params{"from": from.UTC().Format("2006-01-02 15:04:05.000Z")}))
	}
	if !to.IsZero() {
		q = q.AndWhere(dbx.NewExp("kickoff < {:to}", dbx.Params{"to": to.UTC().Format("2006-01-02 15:04:05.000Z")}))
	}
	var recs []*core.Record
	if err := q.OrderBy("kickoff ASC", "num ASC").All(&recs); err != nil {
		return nil, err
	}
	// The pool start date can't go into the query per season; drop what
	// kicked off before it (unless tipped).
	out := recs[:0]
	for _, m := range recs {
		if s.Has(m) {
			out = append(out, m)
		}
	}
	return out, nil
}

// Users returns the ids of everyone whose scope reaches a season through a
// pool (the "in a pool of this season" audience).
func Users(app core.App, tournamentID string) (map[string]bool, error) {
	out := map[string]bool{}
	lgs, err := app.FindRecordsByFilter("pools",
		"tournaments ~ {:t}", "", 0, 0, map[string]any{"t": tournamentID})
	if err != nil {
		return nil, err
	}
	for _, lg := range lgs {
		if lg.GetString("inviteCode") == globalInviteCode {
			continue
		}
		mems, err := app.FindRecordsByFilter("pool_members",
			"pool = {:p}", "", 0, 0, map[string]any{"p": lg.Id})
		if err != nil {
			continue
		}
		for _, m := range mems {
			out[m.GetString("user")] = true
		}
	}
	return out, nil
}
