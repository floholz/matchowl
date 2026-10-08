package h2h

import (
	"log"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/pools"
)

// The next round's pairings stay put when the roster changes. Until a
// round opens its pairings are only a preview, but members plan their save
// calls and bans against it, so a join or leave must not re-rotate the
// whole pool. Instead the pool keeps the preview (pools.h2hNext) and a
// roster change patches it:
//
//   - join: the newcomer takes the Ghost's place, or plays the Ghost when
//     there is none.
//   - leave: a Ghost duel just goes; otherwise whoever played the Ghost
//     steps in for the leaver (the Ghost goes), or, without a Ghost, the
//     Ghost does.
//
// The stored preview is for one ordinal; once that round opens it is
// spent and the following round rotates from the roster as usual.

// stored is the h2hNext payload.
type stored struct {
	Ordinal  int    `json:"ordinal"`
	Pairings []Pair `json:"pairings"`
}

// upcoming is the pairing of the round with the given ordinal before it
// opens: the stored preview when it is for that round and still covers
// exactly the roster, else the rotation.
func upcoming(app core.App, lg *core.Record, ordinal int) []Pair {
	members := roster(app, lg.Id)
	var s stored
	if err := lg.UnmarshalJSONField("h2hNext", &s); err == nil && s.Pairings != nil &&
		s.Ordinal == ordinal && covers(s.Pairings, members) {
		return s.Pairings
	}
	return Pairings(members, ordinal)
}

// covers reports whether the pairings seat exactly these members, once each.
func covers(ps []Pair, members []string) bool {
	want := make(map[string]bool, len(members))
	for _, m := range members {
		want[m] = true
	}
	seen := 0
	for _, p := range ps {
		for _, u := range []string{p.A, p.B} {
			if u == Ghost {
				continue
			}
			if !want[u] {
				return false
			}
			delete(want, u)
			seen++
		}
	}
	return seen == len(members)
}

// withJoined seats a new member: in the Ghost's place, or against the Ghost.
func withJoined(ps []Pair, uid string) []Pair {
	out := append([]Pair(nil), ps...)
	for i, p := range out {
		if p.B == Ghost {
			out[i].B = uid
			return out
		}
	}
	return append(out, Pair{A: uid, B: Ghost})
}

// withLeft unseats a member: their Ghost duel goes; else the Ghost's
// opponent takes their seat (and the Ghost duel goes); else the Ghost does.
func withLeft(ps []Pair, uid string) []Pair {
	out := append([]Pair(nil), ps...)
	at, ghostAt := -1, -1
	for i, p := range out {
		if p.A == uid || p.B == uid {
			at = i
		}
		if p.B == Ghost {
			ghostAt = i
		}
	}
	switch {
	case at < 0:
		return out
	case at == ghostAt:
		return append(out[:at], out[at+1:]...)
	case ghostAt >= 0:
		sub := out[ghostAt].A
		if out[at].A == uid {
			out[at].A = sub
		} else {
			out[at].B = sub
		}
		return append(out[:ghostAt], out[ghostAt+1:]...)
	default:
		if out[at].A == uid {
			out[at].A, out[at].B = out[at].B, Ghost // keep the Ghost on the B side
		} else {
			out[at].B = Ghost
		}
		return out
	}
}

// registerRosterHooks patches a head-to-head pool's stored preview around
// every membership change (joins, leaves, removals, deleted accounts).
func registerRosterHooks(app core.App) {
	patch := func(e *core.RecordEvent, apply func([]Pair, string) []Pair) error {
		lg, err := e.App.FindRecordById("pools", e.Record.GetString("pool"))
		if err != nil || lg.GetString("mode") != pools.ModeH2H {
			return e.Next()
		}
		rows, err := roundRows(e.App, lg.Id)
		if err != nil {
			return e.Next()
		}
		ordinal := len(rows)
		before := upcoming(e.App, lg, ordinal) // the roster as it stands
		if err := e.Next(); err != nil {
			return err
		}
		lg.Set("h2hNext", stored{Ordinal: ordinal, Pairings: apply(before, e.Record.GetString("user"))})
		if err := e.App.Save(lg); err != nil {
			log.Printf("[h2h] keep next pairings for pool %s: %v", lg.Id, err)
		}
		return nil
	}
	app.OnRecordCreate("pool_members").BindFunc(func(e *core.RecordEvent) error {
		return patch(e, withJoined)
	})
	app.OnRecordDelete("pool_members").BindFunc(func(e *core.RecordEvent) error {
		return patch(e, withLeft)
	})
}
